package fswiki_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/entooone/go-fswiki"
)

func TestFormatDocumentWithOption(t *testing.T) {
	cases := []struct {
		beforePath string
		afterPath  string
		option     fswiki.FormatOption
	}{
		{
			beforePath: "table.fswiki",
			afterPath:  "table_right_nospace.fswiki",
			option: fswiki.FormatOption{
				TableAlign:                  fswiki.TableAlignRight,
				TableInsertSpaceToEndOfCell: false,
			},
		},
		{
			beforePath: "table.fswiki",
			afterPath:  "table_right_space.fswiki",
			option: fswiki.FormatOption{
				TableAlign:                  fswiki.TableAlignRight,
				TableInsertSpaceToEndOfCell: true,
			},
		},
		{
			beforePath: "table.fswiki",
			afterPath:  "table_left_space.fswiki",
			option: fswiki.FormatOption{
				TableAlign:                  fswiki.TableAlignLeft,
				TableInsertSpaceToEndOfCell: true,
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.afterPath, func(t *testing.T) {
			srcpath := filepath.Join("testdata", c.beforePath)
			srcbuf, err := os.ReadFile(srcpath)
			if err != nil {
				t.Errorf("cannot open %q", srcpath)
			}

			wantpath := filepath.Join("testdata", c.afterPath)
			wantbuf, err := os.ReadFile(wantpath)
			if err != nil {
				t.Errorf("cannot open %q", wantpath)
			}

			got, err := fswiki.FormatDocumentWithOption(bytes.NewReader(srcbuf), c.option)
			if err != nil {
				t.Errorf("cannot format %q", srcpath)
			}

			if !bytes.Equal(got, wantbuf) {
				t.Errorf("got %q, want %q", got, wantbuf)
			}
		})
	}
}
