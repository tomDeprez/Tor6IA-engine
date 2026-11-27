package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// The engine drives llama.cpp, which does the tensor math and already knows how to place
// the layers of a model on several machines. One build is pinned, with its checksum:
// what is downloaded is exactly what was tested.
const llamaBuild = "b12810"

var llamaBuilds = map[string]struct{ file, sha256 string }{
	"windows/amd64": {"llama-b12810-bin-win-cpu-x64.zip", "845e25154ae4051659cf7363908a663cf15f99c57f8d5157d0f38f468acf660d"},
	"linux/amd64":   {"llama-b12810-bin-ubuntu-x64.tar.gz", "63b94be37dcc34cb4933554d4f998e9649841b23da7ffc71d1f66d63762464d7"},
	"linux/arm64":   {"llama-b12810-bin-ubuntu-arm64.tar.gz", "993d6b073e097bc987d2a34dbc49bab47f79eb2470339c15e34aba3f28aa44d9"},
	"darwin/arm64":  {"llama-b12810-bin-macos-arm64.tar.gz", "911e158dfc97d386b24101543dc9f7be50db09220f8ed80a63e1ddc0b7e119f7"},
	"darwin/amd64":  {"llama-b12810-bin-macos-x64.tar.gz", "03a1859a8e47944383f33d950e3cdb49833fe19f7b7d0bf8a72009502e62fd59"},
}

func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func homeDir() (string, error) {
	if d := os.Getenv("T6E_HOME"); d != "" {
		return d, nil
	}
	base, err := os.UserCacheDir()
	return filepath.Join(base, "tor6ia-engine"), err
}

// ensureLlama returns the folder that holds the llama.cpp programs, downloading them the first time.
func ensureLlama() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, "llama.cpp", llamaBuild)
	if _, err := os.Stat(filepath.Join(root, ".ok")); err != nil {
		build, known := llamaBuilds[runtime.GOOS+"/"+runtime.GOARCH]
		if !known {
			return "", fmt.Errorf("no llama.cpp build is pinned for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
		fmt.Printf("Downloading llama.cpp %s (%s)...\n", llamaBuild, build.file)
		os.RemoveAll(root)
		if err := os.MkdirAll(root, 0o755); err != nil {
			return "", err
		}
		archive := filepath.Join(root, build.file)
		url := "https://github.com/ggml-org/llama.cpp/releases/download/" + llamaBuild + "/" + build.file
		if err := downloadChecked(url, archive, build.sha256); err != nil {
			return "", err
		}
		if strings.HasSuffix(archive, ".zip") {
			err = unzip(archive, root)
		} else {
			err = untar(archive, root)
		}
		if err != nil {
			return "", fmt.Errorf("unpacking llama.cpp: %w", err)
		}
		os.Remove(archive)
		if err := os.WriteFile(filepath.Join(root, ".ok"), nil, 0o644); err != nil {
			return "", err
		}
	}
	// The programs are at the top of the Windows archive and one folder down in the others.
	var dir string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Name() == exe("llama-server") {
			dir = filepath.Dir(p)
		}
		return nil
	})
	if dir == "" {
		return "", errors.New("llama-server is missing from the llama.cpp download: delete " + root + " and try again")
	}
	return dir, nil
}

func downloadChecked(url, dest, wantSHA string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, sum), resp.Body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != wantSHA {
		os.Remove(dest)
		return fmt.Errorf("checksum of %s is %s, expected %s: the download was refused", filepath.Base(dest), got, wantSHA)
	}
	return nil
}

// inside returns where an archive entry goes, refusing names that would leave the folder.
func inside(root, name string) (string, error) {
	p := filepath.Join(root, name)
	if rel, err := filepath.Rel(root, p); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q leaves the folder", name)
	}
	return p, nil
}

func writeFile(p string, r io.Reader, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func unzip(archive, root string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close()
	for _, e := range z.File {
		p, err := inside(root, e.Name)
		if err != nil {
			return err
		}
		if e.FileInfo().IsDir() {
			continue
		}
		r, err := e.Open()
		if err != nil {
			return err
		}
		err = writeFile(p, r, 0o755)
		r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func untar(archive, root string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		p, err := inside(root, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeReg:
			if err := writeFile(p, tr, fs.FileMode(h.Mode)&0o777); err != nil {
				return err
			}
		case tar.TypeSymlink: // versioned library names: libllama.so -> libllama.so.0
			if filepath.IsAbs(h.Linkname) || strings.Contains(h.Linkname, "..") {
				return fmt.Errorf("archive link %q leaves the folder", h.Name)
			}
			os.MkdirAll(filepath.Dir(p), 0o755)
			os.Remove(p)
			if err := os.Symlink(h.Linkname, p); err != nil {
				return err
			}
		}
	}
}

// resolveModel turns what the user typed into a GGUF file: a path, or the name of a model
// Ollama already downloaded (gemma4:12b), so that nothing is downloaded twice.
func resolveModel(name string) (string, error) {
	if _, err := os.Stat(name); err == nil {
		return filepath.Abs(name) // llama.cpp runs from its own folder: a relative path would not be found
	}
	family, tag, ok := strings.Cut(name, ":")
	if !ok {
		tag = "latest"
	}
	models := os.Getenv("OLLAMA_MODELS")
	if models == "" {
		home, _ := os.UserHomeDir()
		models = filepath.Join(home, ".ollama", "models")
	}
	raw, err := os.ReadFile(filepath.Join(models, "manifests", "registry.ollama.ai", "library", family, tag))
	if err != nil {
		return "", fmt.Errorf("model %q is neither a GGUF file nor a model Ollama has: run \"ollama pull %s\" or give the path of a .gguf file", name, name)
	}
	var manifest struct {
		Layers []struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", err
	}
	for _, l := range manifest.Layers {
		if strings.HasSuffix(l.MediaType, "image.model") {
			return filepath.Join(models, "blobs", strings.Replace(l.Digest, ":", "-", 1)), nil
		}
	}
	return "", fmt.Errorf("Ollama's copy of %q has no model file", name)
}
