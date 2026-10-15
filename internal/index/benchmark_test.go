package index

import (
	"fmt"
	"knodexa/internal/model"
	"knodexa/internal/storage"
	"path/filepath"
	"testing"
)

func benchmarkIndex(b *testing.B) (*Indexer, *storage.Storage) {
	b.Helper()
	store := storage.NewStorage(b.TempDir())
	if _, err := store.Init("Bench", ""); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		if err := store.SaveTask(&model.TaskData{ID: fmt.Sprintf("T-%03d", i), Title: "Task", Goal: "Preserve continuity", Constraints: []string{"Preserve API"}, TouchesFiles: []string{"auth.go"}}); err != nil {
			b.Fatal(err)
		}
		if err := store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: fmt.Sprintf("L-%03d", i), SourceTaskID: "source", Title: "Reject bool timestamps", Status: model.StatusVerified, RelatedFiles: []string{"auth.go"}}, Body: "Reject boolean values when validating integer timestamps."}); err != nil {
			b.Fatal(err)
		}
	}
	db, err := Open(filepath.Join(store.KnodexaDir(), "cache.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })
	idx := NewIndexer(db, store)
	if err := idx.Sync(); err != nil {
		b.Fatal(err)
	}
	return idx, store
}

func BenchmarkSyncUnchanged(b *testing.B) {
	idx, _ := benchmarkIndex(b)
	var before, after int64
	if err := idx.db.sqlDB.QueryRow("SELECT total_changes()").Scan(&before); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := idx.Sync(); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if err := idx.db.sqlDB.QueryRow("SELECT total_changes()").Scan(&after); err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(after-before)/float64(b.N), "rows-written/op")
}

func BenchmarkSyncOneChangedTask(b *testing.B) {
	idx, store := benchmarkIndex(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := store.SaveTask(&model.TaskData{ID: "T-000", Title: fmt.Sprintf("Edited %d", i), Goal: "Preserve continuity", Constraints: []string{"NEW"}}); err != nil {
			b.Fatal(err)
		}
		if err := idx.Sync(); err != nil {
			b.Fatal(err)
		}
	}
}
