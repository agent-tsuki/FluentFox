package content

import (
	"encoding/json"
	"os"
	"testing"
)

// Every chapter must have enough of each kind for its exercise to ask them all.
func TestShippedExercisesAreValid(t *testing.T) {
	raw, err := os.ReadFile("../../content/exercises/n5.json")
	if err != nil {
		t.Fatal(err)
	}
	var file ExerciseFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if err := CheckExerciseFile(&file); err != nil {
		t.Fatal(err)
	}
	for i := range file.Chapters {
		c := &file.Chapters[i]
		for name, n := range map[string]int{"true/false": len(c.TrueFalse), "transform": len(c.Transforms), "word": len(c.Words), "order": len(c.Orders), "listen": len(c.Listens), "reply": len(c.Replies), "reading": len(c.Readings)} {
			if n < 2 {
				t.Errorf("%s has %d %s exercises", c.Slug, n, name)
			}
		}
	}
}
