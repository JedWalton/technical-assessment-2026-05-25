package batch

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JedWalton/technical-assessment-2026-05-25/internal/order"
)

var csvHeaderRow = []string{"customer_number", "address", "postcode", "placed_at"}

// FileWriter writes batches of orders to CSV files in a directory using an
// atomic rename so consumers never read partial files.
var _ Writer = (*FileWriter)(nil)

type FileWriter struct {
	Dir string
}

// NewFileWriter returns a writer that creates files under dir.
func NewFileWriter(dir string) *FileWriter {
	return &FileWriter{Dir: dir}
}

// Write persists orders to a new CSV file. Returns nil without creating a file
// when orders is empty. Respects ctx cancellation.
func (w *FileWriter) Write(ctx context.Context, orders []order.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(orders) == 0 {
		return nil
	}

	tmpPath, finalPath, err := w.newPaths()
	if err != nil {
		return err
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create batch file: %w", err)
	}

	writeErr := func() error {
		cw := csv.NewWriter(f)
		if err := cw.Write(csvHeaderRow); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, o := range orders {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := cw.Write([]string{
				o.CustomerNumber,
				o.Address,
				o.Postcode,
				o.PlacedAt.UTC().Format(time.RFC3339),
			}); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return fmt.Errorf("flush csv: %w", err)
		}
		return nil
	}()

	if err := writeErr; err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close batch file: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename batch file: %w", err)
	}
	return nil
}

func (w *FileWriter) newPaths() (tmpPath, finalPath string, err error) {
	id, err := randomHex(4)
	if err != nil {
		return "", "", fmt.Errorf("generate file id: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	base := fmt.Sprintf("orders-%s-%s.csv", stamp, id)
	tmpPath = filepath.Join(w.Dir, base+".tmp")
	finalPath = filepath.Join(w.Dir, base)
	return tmpPath, finalPath, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
