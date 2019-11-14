package shifters

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type MixerShifter struct{}

func (s MixerShifter) Shift(r Replacers) error {
	err := filepath.Walk("mixer", replace(r))
	if err != nil {
		return err
	}

	return nil
}

func replace(r Replacers) func(path string, info os.FileInfo, err error) error {
	return func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		fmt.Printf("\nReplacing in file: %s ", path)

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		for _, replacer := range r {
			fmt.Print(".")
			content = bytes.ReplaceAll(content, []byte(replacer.From), []byte(replacer.To))
		}

		fmt.Print("DONE.")

		err = ioutil.WriteFile(path, []byte(content), 0)
		if err != nil {
			return err
		}

		return nil
	}
}
