//go:build windows

package dialog

import (
	"io"
	"os"

	"github.com/harry1453/go-common-file-dialog/cfd"
	"github.com/harry1453/go-common-file-dialog/cfdutil"
)

// SaveDorks opens the Windows Explorer save dialog and copies dorks to the chosen path.
func SaveDorks(srcPath string) (dest string, cancelled bool, err error) {
	dest, err = cfdutil.ShowSaveFileDialog(cfd.DialogConfig{
		Title:            "Enregistrer les dorks",
		Role:             "letter_dorks",
		FileName:         "dorks.txt",
		DefaultExtension: "txt",
		FileFilters: []cfd.FileFilter{
			{DisplayName: "Fichiers texte (*.txt)", Pattern: "*.txt"},
			{DisplayName: "Tous les fichiers (*.*)", Pattern: "*.*"},
		},
	})
	if err == cfd.ErrorCancelled {
		return "", true, nil
	}
	if err != nil {
		return "", false, err
	}
	if err := copyFile(srcPath, dest); err != nil {
		return "", false, err
	}
	return dest, false, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
