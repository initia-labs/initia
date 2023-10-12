package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/spf13/cobra"
)

const (
	flagVerifierURL = "verifier-url"
)

const defaultVerifierURL = "https://api.initia.tech/contracts/verify" // FIXME: set to real url

const (
	manifestFilename = "Move.toml"
	sourcesDirname   = "sources"
	srcExtension     = ".move"
)

type verifyConfig struct {
	PackagePath string
	VerifierURL *url.URL
	ChainID     string
}

func getVerifyConfig(cmd *cobra.Command) (vc *verifyConfig, err error) {
	vc = &verifyConfig{}

	vc.PackagePath, err = cmd.Flags().GetString(flagPackagePath)
	if err != nil {
		return nil, err
	}

	urlStr, err := cmd.Flags().GetString(flagVerifierURL)
	if err != nil {
		return nil, err
	}

	vc.VerifierURL, err = url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	vc.ChainID, err = cmd.Flags().GetString(flags.FlagChainID)
	if err != nil {
		return nil, err
	}
	if vc.ChainID == "" {
		return nil, fmt.Errorf("chain id is required")
	}

	return vc, err
}

// send the contract to the verifier
func verifyContract(vc verifyConfig) (err error) {
	packageBuf := bytes.NewBuffer([]byte{})

	// create zip buffer for package zipping
	zipW := zip.NewWriter(packageBuf)

	manifestPath := path.Join(vc.PackagePath, manifestFilename)
	err = zipFile(zipW, vc.PackagePath, manifestPath)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("failed to zip %s", manifestPath)))
	}

	err = filepath.WalkDir(path.Join(vc.PackagePath, sourcesDirname),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return errors.Wrap(err, "failed to walk dir")
			}

			if strings.HasSuffix(entry.Name(), srcExtension) {
				if err := zipFile(zipW, vc.PackagePath, path); err != nil {
					return errors.Wrap(err, "failed to zip move source files")
				}
			}

			return nil
		})
	if err != nil {
		return errors.Wrap(err, "failed to aggregate move source files")
	}
	zipW.Close()

	// create multipart post request body
	mpBuf := bytes.NewBuffer([]byte{})
	mpW := multipart.NewWriter(mpBuf)

	fieldW, err := mpW.CreateFormField("chainId")
	if err != nil {
		return errors.Wrap(err, "failed to create form for address")
	}
	_, err = fieldW.Write([]byte(vc.ChainID))
	if err != nil {
		return errors.Wrap(err, "failed to write form for address")
	}

	fieldW, err = mpW.CreateFormField("package")
	if err != nil {
		return errors.Wrap(err, "failed to create form for package")
	}
	_, err = fieldW.Write([]byte(base64.RawStdEncoding.EncodeToString(packageBuf.Bytes())))
	if err != nil {
		return errors.Wrap(err, "failed to write form package")
	}
	mpW.Close()

	// create post request
	req, err := http.NewRequest(http.MethodPost, vc.VerifierURL.String(), mpBuf)
	if err != nil {
		return errors.Wrap(err, "failed to create verify request")
	}
	req.Header.Set("Content-Type", mpW.FormDataContentType())
	req.Header.Set("Accept-Encoding", "deflate")

	res, err := http.DefaultClient.Do(req)
	if err != nil || (res.StatusCode/100 != 2) {
		if err != nil {
			return errors.Wrap(err, "failed to post to verifier")
		}

		msg, err := io.ReadAll(res.Body)
		if err != nil {
			msg = []byte(res.Status)
		}
		return fmt.Errorf("failed to post to verifier: %s", string(msg))
	}

	return err
}

func addMoveVerifyFlags(cmd *cobra.Command, isVerifyCmd bool) {
	cmd.Flags().String(flagVerifierURL, defaultVerifierURL, "URL of the verifier")
	if isVerifyCmd {
		cmd.Flags().StringP(flagPackagePath, flagPackagePathShorthand, defaultPackagePath, "Path to a package which the command should be run with respect to")
	}
}

func zipFile(zipWriter *zip.Writer, packagePath, fpath string) error {
	relpath, err := filepath.Rel(packagePath, fpath)
	if err != nil {
		return errors.Wrap(err, "failed to get relative path")
	}
	w, err := zipWriter.Create(relpath)
	if err != nil {
		return errors.Wrap(err, "failed to create file on zip")
	}
	b, err := os.ReadFile(fpath)
	if err != nil {
		return errors.Wrap(err, "failed to read file to zip")
	}
	_, err = w.Write(b)
	if err != nil {
		return errors.Wrap(err, "failed to write file on zip")
	}
	return nil
}
