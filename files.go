package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func (c *QVSClient) fsReq(function string, query string, form url.Values) (*http.Response, error) {
	reqURL := fmt.Sprintf("%s%s?func=%s&sid=%s%s", c.QtsURL, QTSFileStation, function, c.SessionID, query)

	var req *http.Request
	if form == nil {
		req, _ = http.NewRequest("POST", reqURL, nil)
	} else {
		req, _ = http.NewRequest("POST", reqURL, strings.NewReader(form.Encode()))
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	client := &http.Client{
		Jar: c.CookieJar,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error making request, HTTP status code: %d", resp.StatusCode)
	}

	return resp, err
}

func (c *QVSClient) ListDir(qtsPath string) ([]ListFile, error) {
	form := url.Values{}
	form.Add("path", qtsPath)
	form.Add("start", "0")
	form.Add("limit", "500")
	form.Add("sort", "natural")
	form.Add("dir", "ASC")

	resp, err := c.fsReq("get_list", "", form)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	type response struct {
		Datas []ListFile `json:"datas"`
	}

	var jsonData response
	err = json.Unmarshal(data, &jsonData)

	return jsonData.Datas, nil
}

func (c *QVSClient) CreateDir(destDir string) error {
	destPath := path.Dir(destDir)
	destFolder := path.Base(destDir)

	form := url.Values{}
	form.Add("dest_path", destPath)
	form.Add("dest_folder", destFolder)

	_, err := c.fsReq("createdir", "", form)
	if err != nil {
		return err
	}

	return nil
}

func (c *QVSClient) CopyFile(srcPath string, destPath string) error {
	srcFile := path.Base(srcPath)
	srcDir := path.Dir(srcPath)
	destDir := path.Dir(destPath)

	form := url.Values{}
	form.Add("source_total", "1")
	form.Add("mode", "0")
	form.Add("source_file", srcFile)
	form.Add("source_path", srcDir)
	form.Add("dest_path", destDir)

	_, err := c.fsReq("copy", "", form)
	if err != nil {
		return err
	}
	return nil
}

func (c *QVSClient) RenameFile(srcPath, srcName, destName string) error {
	form := url.Values{}
	form.Add("path", srcPath)
	form.Add("source_name", srcName)
	form.Add("dest_name", destName)

	_, err := c.fsReq("rename", "", form)
	if err != nil {
		return err
	}
	return nil
}

func (c *QVSClient) UploadFile(srcFile *os.File, destPath string) error {
	destDir := path.Dir(destPath)
	qtsPath := strings.Replace(destPath, "/", "-", -1)
	reqURL := fmt.Sprintf("%s%s?sid=%s&func=upload&type=standard&dest_path=%s&overwrite=1&progress=%s", c.QtsURL, QTSFileStation, c.SessionID, destDir, qtsPath)

	client := &http.Client{
		Jar: c.CookieJar,
	}

	values := map[string]io.Reader{
		"data": srcFile,
	}
	resp, err := upload(client, reqURL, values)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error uploading file")
	}

	return nil
}

func upload(client *http.Client, url string, values map[string]io.Reader) (*http.Response, error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	var err error
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return nil, err
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return nil, err
			}
		}
		if _, err := io.Copy(fw, r); err != nil {
			return nil, err
		}

	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return nil, err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	return client.Do(req)
}
