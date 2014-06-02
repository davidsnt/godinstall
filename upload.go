package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"code.google.com/p/go.crypto/openpgp"
)

type UploadSessioner interface {
	SessionID() string
	SessionURL() string
	AddChanges(*DebChanges)
	Changes() *DebChanges
	AddFile(*ChangesFile) error
	Files() map[string]*ChangesFile
	HandleReq(w http.ResponseWriter, r *http.Request)
	Close()
}

type uploadSession struct {
	SessionId  string // Name of the session
	dir        string // Temporary directory for storage
	keyRing    openpgp.KeyRing
	requireSig bool
	changes    *DebChanges
}

func NewUploadSessioner(a *AptServer) UploadSessioner {
	var s uploadSession
	s.SessionId = uuid.New()
	s.keyRing = a.pubRing
	s.dir = a.TmpDir + "/" + s.SessionId

	os.Mkdir(s.dir, os.FileMode(0755))

	a.sessMap.Set(s.SessionId, &s)
	go pathHandle(a.sessMap, s.SessionId, a.TTL)

	return &s
}

func (s *uploadSession) SessionID() string {
	return s.SessionId
}

func (s *uploadSession) SessionURL() string {
	return "/package/upload/" + s.SessionId
}

func (s *uploadSession) HandleReq(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		{
			j := UploadSessionToJSON(s)
			w.Write(j)
			return
		}
	case "PUT", "POST":
		{
			// This response code will depend on upload status
			w.WriteHeader(202)
			w.Write([]byte("Upload some junk"))
			return
		}
	default:
		{
			http.Error(w, "unknown method", http.StatusInternalServerError)
			return
		}
	}
}

func (s *uploadSession) Close() {
	os.Remove(s.dir)
}

func pathHandle(sessMap *SafeMap, s string, timeout time.Duration) {
	time.Sleep(timeout)
	c := sessMap.Get(s)
	if c != nil {
		switch sess := c.(type) {
		case *uploadSession:
			log.Println("Close session")
			sess.Close()
			sessMap.Delete(s)
		default:
			log.Println("Shouldn't get here")
		}
	} else {
		log.Println("Didn't find session")
	}
}

func (s *uploadSession) AddChanges(c *DebChanges) {
	s.changes = c
}

func (s *uploadSession) Changes() *DebChanges {
	return s.changes
}

func (s *uploadSession) AddFile(f *ChangesFile) (err error) {
	// Check that there is an upload slot
	f, ok := s.changes.Files[f.Filename]
	if !ok {
		return errors.New("File not listed in upload set")
	}

	if f.Uploaded {
		return errors.New("File already uploaded")
	}

	log.Println(f.data)

	md5er := md5.New()
	sha1er := sha1.New()
	sha256er := sha256.New()
	hasher := io.MultiWriter(md5er, sha1er, sha256er)
	tee := io.TeeReader(f.data, hasher)
	tmpfile, err := os.Create("/tmp/upload/" + f.Filename)
	if err != nil {
		return errors.New("Upload failed: " + err.Error())
	}

	_, err = io.Copy(tmpfile, tee)
	if err != nil {
		return errors.New("Upload failed: " + err.Error())
	}

	md5 := base64.StdEncoding.EncodeToString(md5er.Sum(nil))
	sha1 := base64.StdEncoding.EncodeToString(sha1er.Sum(nil))
	sha256 := base64.StdEncoding.EncodeToString(sha256er.Sum(nil))

	log.Println("md5: " + string(md5))
	log.Println("sha1 " + string(sha1))
	log.Println("sha256 " + string(sha256))

	return
}

func (s *uploadSession) Files() map[string]*ChangesFile {
	return s.changes.Files
}

func UploadSessionToJSON(s UploadSessioner) []byte {
	resp := struct {
		SessionId  string
		SessionURL string
		Changes    DebChanges
	}{
		s.SessionID(),
		s.SessionURL(),
		*s.Changes(),
	}
	j, _ := json.Marshal(resp)
	return j
}