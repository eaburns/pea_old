// © 2020 the Pea Authors under the MIT license. See AUTHORS for the list of authors.

package mod

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEmptyModule(t *testing.T) {
	root, err := newFS(nil)
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	m, err := Load(root, "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(m.SrcFiles) > 0 {
		t.Errorf("len(m.SrcFiles)=%d, want 0", len(m.SrcFiles))
	}
}

func TestSourceFileModule(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo.pea", body: ""},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	m, err := Load(filepath.Join(root, "foo.pea"), "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	want := []string{
		filepath.Join(root, "foo.pea"),
	}
	if !reflect.DeepEqual(m.SrcFiles, want) {
		t.Errorf("m.SrcFiles=%v, want %v", m.SrcFiles, want)
	}
}

func TestSourceFileNotFound(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo.pea", body: ""},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	_, err = Load(filepath.Join(root, "nothing.pea"), "foo")
	if err == nil {
		t.Fatalf("Load() succeeded, wanted an error")
	}
}

func TestSourceDirModule(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo.pea", body: ""},
		{path: "bar.pea", body: ""},
		{path: "baz.pea", body: ""},
		{path: "zzz.go", body: ""},
		{path: "qux.go", body: ""},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	m, err := Load(root, "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{
		filepath.Join(root, "bar.pea"),
		filepath.Join(root, "baz.pea"),
		filepath.Join(root, "foo.pea"),
	}
	if !reflect.DeepEqual(m.SrcFiles, want) {
		t.Errorf("m.SrcFiles=%v, want %v", m.SrcFiles, want)
	}
	wantGo := []string{
		filepath.Join(root, "qux.go"),
		filepath.Join(root, "zzz.go"),
	}
	if !reflect.DeepEqual(m.GoSrcFiles, wantGo) {
		t.Errorf("m.GoSrcFiles=%v, want %v", m.GoSrcFiles, wantGo)
	}
}

func TestSourceDirIgnoreNonPeaFiles(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo.pea", body: ""},
		{path: ".gitignore", body: ""},
		{path: "blah", body: ""},
		{path: "something.pea_something", body: ""},
		{path: "pea", body: ""},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	m, err := Load(root, "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{
		filepath.Join(root, "foo.pea"),
	}
	if !reflect.DeepEqual(m.SrcFiles, want) {
		t.Errorf("m.SrcFiles=%v, want %v", m.SrcFiles, want)
	}
}

func TestMalformedImport(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo/foo.pea", body: `import "bar"`},
		{path: "bar/bar.pea", body: `import malformed_not_quoted`},
		{path: "baz/baz.pea", body: ``},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	m, err := Load(filepath.Join(root, "foo"), "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := m.LoadDeps(root); err == nil {
		t.Fatalf("LoadDeps succeeded, wanted an error")
	}
}

func TestMissingDep(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo/foo.pea", body: `import "bar"`},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	m, err := Load(filepath.Join(root, "foo"), "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := m.LoadDeps(root); err == nil {
		t.Fatalf("LoadDeps succeeded, wanted an error")
	}
}

func TestLoadDeps(t *testing.T) {
	root, err := newFS([]file{
		{path: "foo/foo.pea", body: `import "bar"`},
		{path: "bar/bar.pea", body: `import "baz"`},
		{path: "baz/baz.pea", body: ``},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	fooDir := filepath.Join(root, "foo")

	foo, err := Load(fooDir, "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := foo.LoadDeps(root); err != nil {
		t.Fatalf("LoadDeps failed: %v", err)
	}
	if len(foo.Deps) != 1 {
		t.Fatalf("len(foo.Deps)=%d, want 1", len(foo.Deps))
	}

	bar := foo.Deps[0]
	if bar.ModPath != "bar" {
		t.Errorf("bar.ModPath=%v, want bar", bar.ModPath)
	}
	if len(bar.Deps) != 1 {
		t.Fatalf("len(bar.Deps)=%d, want 1", len(bar.Deps))
	}

	baz := bar.Deps[0]
	if baz.ModPath != "baz" {
		t.Errorf("bar.ModPath=%v, want baz", baz.ModPath)
	}
	if len(baz.Deps) != 0 {
		t.Errorf("len(baz.Deps)=%d, want 0", len(baz.Deps))
	}
}

func TestTopologicalDeps(t *testing.T) {
	root, err := newFS([]file{
		{
			path: "foo/foo.pea",
			body: `
				import "bar"
				import "baz"
			`,
		},
		{path: "bar/bar.pea", body: `import "baz"`},
		{path: "baz/baz.pea", body: ``},
	})
	if err != nil {
		t.Fatalf("newFS failed: %v", err)
	}
	defer rmDirRecur(root)

	fooDir := filepath.Join(root, "foo")

	foo, err := Load(fooDir, "foo")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := foo.LoadDeps(root); err != nil {
		t.Fatalf("LoadDeps failed: %v", err)
	}

	sorted := TopologicalDeps([]*Mod{foo})
	if len(sorted) != 3 {
		t.Fatalf("len(sorted)=%d, want 3", len(sorted))
	}
	if sorted[0].ModPath != "baz" {
		t.Errorf("sorted[0].ModPath=%v, want baz", sorted[0].ModPath)
	}
	if sorted[1].ModPath != "bar" {
		t.Errorf("sorted[1].ModPath=%v, want bar", sorted[1].ModPath)
	}
	if sorted[2].ModPath != "foo" {
		t.Errorf("sorted[2].ModPath=%v, want foo", sorted[2].ModPath)
	}
}

type file struct {
	path string
	body string
}

// newFS creates the files in a root temporary directory.
// It returns the root directory or an error.
func newFS(files []file) (root string, err error) {
	if root, err = ioutil.TempDir("", "pea_mod_test"); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			rmDirRecur(root)
		}
	}()
	for _, file := range files {
		if err := mkDirRecur(root, filepath.Dir(file.path)); err != nil {
			return "", err
		}
		f, err := os.Create(filepath.Join(root, file.path))
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(f, file.body); err != nil {
			return "", err
		}
		if err := f.Close(); err != nil {
			return "", err
		}
	}
	return root, nil
}

func mkDirRecur(root, dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	if err := mkDirRecur(root, filepath.Dir(dir)); err != nil {
		return err
	}
	return os.Mkdir(filepath.Join(root, dir), os.ModePerm)
}

func rmDirRecur(root string) error {
	f, err := os.Open(root)
	if err != nil {
		return err
	}
	defer f.Close()

	fileInfos, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	for _, fileInfo := range fileInfos {
		var err error
		if fileInfo.IsDir() {
			err = rmDirRecur(fileInfo.Name())
		} else {
			err = os.Remove(fileInfo.Name())
		}
		if err != nil {
			return err
		}
	}
	return os.Remove(root)
}
