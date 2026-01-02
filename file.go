package puff

import (
	"mime/multipart"
	"os"
)

type File struct {
	Name          string
	Size          int64
	MultipartFile multipart.File
}

func (f *File) SaveTo(filepath ...string) (n int, err error) {
	fp := ""
	if len(filepath) == 0 {
		fp = f.Name
	} else {
		fp = filepath[0]
	}
	data := make([]byte, f.Size)
	_, err = f.MultipartFile.Read(data)
	if err != nil {
		return
	}
	defer f.MultipartFile.Close()
	file, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return 0, err
	}
	n, err = file.Write(data)
	return
}
