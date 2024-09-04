package logger

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"gitlab.com/linkinlog/cloudKV/store"
)

const Table = "transactions"

func NewPostgresTransactionLogger(config PostgresDBParams) (*PostgresTransactionLogger, error) {
	connStr := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable",
		config.host, config.dbName, config.user, config.password)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	logger := &PostgresTransactionLogger{db: db}

	exists, err := logger.verifyTableExists(Table)
	if err != nil {
		return nil, fmt.Errorf("failed to verify table: %w", err)
	}
	if !exists {
		if err := logger.createTxTable(); err != nil {
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	return logger, nil
}

type PostgresDBParams struct {
	dbName, host, user, password string
}

type PostgresTransactionLogger struct {
	events chan<- store.Event
	errors chan error
	db     *sql.DB
}

func (l *PostgresTransactionLogger) Close() error {
	return l.db.Close()
}

func (l *PostgresTransactionLogger) LogPut(key, value string) error {
	l.events <- store.Event{EventType: store.EventPut, Key: key, Value: value}

	return nil
}

func (l *PostgresTransactionLogger) LogDelete(key string) error {
	l.events <- store.Event{EventType: store.EventDelete, Key: key}

	return nil
}

func (l *PostgresTransactionLogger) Err() <-chan error {
	return l.errors
}

func (l *PostgresTransactionLogger) ReadEvents() (<-chan store.Event, <-chan error) {
	outEvent := make(chan store.Event)
	outError := make(chan error, 1)

	go func() {
		defer close(outEvent)
		defer close(outError)

		query := `select sequence, event_type, key, value from transactions order by sequence`

		rows, err := l.db.Query(query)
		if err != nil {
			outError <- fmt.Errorf("sql query error: %w", err)
			return
		}

		defer rows.Close()

		e := store.Event{}

		for rows.Next() {
			err = rows.Scan(
				&e.Sequence,
				&e.EventType,
				&e.Key,
				&e.Value,
			)
			if err != nil {
				outError <- fmt.Errorf("error reading row: %w", err)
				return
			}

			outEvent <- e
		}

		if err := rows.Err(); err != nil {
			outError <- fmt.Errorf("error reading rows: %w", err)
			return
		}
	}()

	return outEvent, outError
}

func (l *PostgresTransactionLogger) Run() {
	events := make(chan store.Event, 16)
	l.events = events

	errs := make(chan error, 1)
	l.errors = errs

	go func() {
		query := `insert into transactions (event_type, key, value) values ($1, $2, $3)`

		for e := range events {
			if _, err := l.db.Exec(
				query,
				e.EventType,
				e.Key,
				e.Value,
			); err != nil {
				errs <- err
			}
		}
	}()
}

func (l *PostgresTransactionLogger) verifyTableExists(table string) (bool, error) {
	if l.db != nil {
		tx, err := l.db.Begin()
		if err != nil {
			return false, err
		}
		row := tx.QueryRow(`SELECT EXISTS (
							   SELECT FROM information_schema.tables
							   WHERE  table_schema = 'public'
							   AND    table_name   = $1
							   );`,
			table)

		if row.Err() != nil {
			return false, row.Err()
		}
		if err := tx.Commit(); err != nil {
			return false, err
		}
	}
	return false, nil
}

func (l *PostgresTransactionLogger) createTxTable() error {
	if l.db != nil {
		tx, err := l.db.Begin()
		if err != nil {
			return err
		}

		createTableQuery := `
CREate table if not exists transactions (
  sequence serial primary key,
  event_type int,
  key text,
  value text
)
`
		if _, err = tx.Exec(createTableQuery); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
