package cli

import (
	"knodexa/internal/model"
	"testing"
)

func TestSupersedeCLIUsesVerifiedReplacement(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"init", "--name=Replacement"}); err != nil {
		t.Fatal(err)
	}
	app, err = NewApp(root)
	if err != nil {
		t.Fatal(err)
	}
	defer app.db.Close()
	for _, id := range []string{"L-old", "L-new"} {
		if err := app.store.SaveLesson(&model.LessonDocument{Frontmatter: model.LessonFrontmatter{ID: id, Status: model.StatusVerified}, Body: "Inspect original content"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Run([]string{"supersede", "L-old", "--by=L-new"}); err == nil {
		t.Fatal("missing reason accepted")
	}
	if err := app.Run([]string{"supersede", "L-old", "--by=L-new", "--reason=Verified interface migration"}); err != nil {
		t.Fatal(err)
	}
	old, err := app.store.ReadLesson("L-old")
	if err != nil {
		t.Fatal(err)
	}
	if old.Frontmatter.SupersededBy != "L-new" || old.Frontmatter.Status != model.StatusSuperseded {
		t.Fatal("CLI did not retire knowledge")
	}
}
