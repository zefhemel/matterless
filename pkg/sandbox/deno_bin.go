package sandbox

import (
	"archive/zip"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/zefhemel/matterless/pkg/config"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
)

// TODO: Add windows versions
var denoDownloadUrls = map[string]string{
	"linux-arm64":  "https://matterless-releases.s3.eu-central-1.amazonaws.com/deno-linux-arm64.zip",
	"linux-amd64":  "https://github.com/denoland/deno/releases/download/v1.8.2/deno-x86_64-unknown-linux-gnu.zip",
	"darwin-arm64": "https://github.com/denoland/deno/releases/download/v1.8.2/deno-aarch64-apple-darwin.zip",
	"darwin-amd64": "https://github.com/denoland/deno/releases/download/v1.8.2/deno-x86_64-apple-darwin.zip",
}

func denoBinPath(config *config.Config) string {
	// TODO: Reenable auto download
	//return fmt.Sprintf("%s/.deno/deno", config.DataDir)
	return "deno"
}

func ensureDeno(config *config.Config) error {
	denoPath := denoBinPath(config)
	buildToGet := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if _, err := os.Stat(denoPath); err != nil {
		url, ok := denoDownloadUrls[buildToGet]
		if !ok {
			return fmt.Errorf("No deno download ready for %s", buildToGet)
		}
		downloadTo := fmt.Sprintf("%s/deno.gz", os.TempDir())
		log.Infof("Downloading deno from %s", url)
		if err := downloadFile(url, downloadTo); err != nil {
			return errors.Wrap(err, "deno download")
		}
		r, err := zip.OpenReader(downloadTo)
		if err != nil {
			return errors.Wrap(err, "open deno zip")
		}
		defer r.Close()

		if len(r.File) != 1 {
			return fmt.Errorf("Expected just one file in zip, but got %d", len(r.File))
		}

		rc, err := r.File[0].Open()
		if err != nil {
			log.Fatal(err)
		}
		if err := os.MkdirAll(path.Dir(denoPath), 0700); err != nil {
			return errors.Wrap(err, "create deno dir")
		}
		ft, err := os.Create(denoPath)
		if err != nil {
			return errors.Wrap(err, "write deno bin")
		}
		_, err = io.Copy(ft, rc)
		if err != nil {
			return errors.Wrap(err, "copy deno bin")
		}
		ft.Close()
		rc.Close()
		os.Chmod(denoPath, 0755)

	}
	return nil
}

func downloadFile(url string, toPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
