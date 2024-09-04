package logger

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gitlab.com/linkinlog/cloudKV/store"
)

func NewFileTransactionLogger(filename string) (*FileTransactionLogger, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return &FileTransactionLogger{file: file}, nil
}

type FileTransactionLogger struct {
	events chan<- store.Event
	errors chan error
	last   store.Sequence
	file   *os.File
}

func (ftl *FileTransactionLogger) Close() error {
	return ftl.file.Close()
}

func (ftl *FileTransactionLogger) LogPut(key, value string) error {
	ftl.events <- store.Event{EventType: store.EventPut, Key: key, Value: value}

	return nil
}

func (ftl *FileTransactionLogger) LogDelete(key string) error {
	ftl.events <- store.Event{EventType: store.EventDelete, Key: key}

	return nil
}

func (ftl *FileTransactionLogger) Err() <-chan error {
	return ftl.errors
}

func (ftl *FileTransactionLogger) Run() {
	events := make(chan store.Event, 16)
	ftl.events = events

	errors := make(chan error, 1)
	ftl.errors = errors

	go func() {
		for e := range events {
			ftl.last++

			_, err := fmt.Fprintf(
				ftl.file,
				"%d\t%d\t%s\t%s\n",
				ftl.last, e.EventType, e.Key, e.Value,
			)
			if err != nil {
				errors <- err
				return
			}
		}
	}()
}

func (ftl *FileTransactionLogger) ReadEvents() (<-chan store.Event, <-chan error) {
	scanner := bufio.NewScanner(ftl.file)
	outEvent := make(chan store.Event)
	outError := make(chan error, 1)

	go func() {
		var e store.Event

		defer close(outEvent)
		defer close(outError)

		for scanner.Scan() {
			line := scanner.Text()

            line = strings.ReplaceAll(line, " ", "_")
			if _, err := fmt.Sscanf(
				line, "%d\t%d\t%s\t%s",
				&e.Sequence, &e.EventType, &e.Key, &e.Value,
			); err != nil {
				outError <- fmt.Errorf("input parse error: %w", err)
				return
			}
            e.Key = strings.ReplaceAll(e.Key, "_", " ")
            e.Value = strings.ReplaceAll(e.Value, "_", " ")

			if ftl.last >= e.Sequence {
				outError <- fmt.Errorf("sequence number error: %d >= %d", ftl.last, e.Sequence)
				return
			}

			ftl.last = e.Sequence

			outEvent <- e
		}

		if err := scanner.Err(); err != nil {
			outError <- fmt.Errorf("transaction log read failure: %w", err)
			return
		}
	}()

	return outEvent, outError
}
