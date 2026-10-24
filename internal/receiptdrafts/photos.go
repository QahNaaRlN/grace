package receiptdrafts

import (
	"github.com/QahNaaRlN/grace/internal/documentdata"

	"crypto/rand"

	"encoding/hex"
	"errors"

	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxPhotoBytes = documentdata.MaxPhotoBytes

func NormalizePhoto(raw []byte) ([]byte, error) { return documentdata.NormalizePhoto(raw) }

type PhotoStore struct{ directory string }

func NewPhotoStore(directory string) (*PhotoStore, error) {
	if !filepath.IsAbs(directory) {
		return nil, errors.New("photo_storage_unavailable")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	st, err := os.Lstat(directory)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("photo_storage_unavailable")
	}
	return &PhotoStore{filepath.Clean(directory)}, nil
}
func (s *PhotoStore) Save(data []byte) (string, error) { return s.save(data, MaxPhotoBytes) }
func (s *PhotoStore) save(data []byte, limit int) (string, error) {
	if len(data) == 0 || len(data) > limit {
		return "", errors.New("invalid_image")
	}
	var token [24]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(token[:])
	f, err := os.OpenFile(filepath.Join(s.directory, id+".jpg"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(filepath.Join(s.directory, id+".jpg"))
		if err != nil {
			return "", err
		}
		return "", closeErr
	}
	return id, nil
}
func (s *PhotoStore) path(id string) (string, error) {
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 24 || len(id) != 48 {
		return "", errors.New("invalid_image_id")
	}
	return filepath.Join(s.directory, id+".jpg"), nil
}
func (s *PhotoStore) Read(id string) ([]byte, error) { return s.read(id, MaxPhotoBytes) }
func (s *PhotoStore) read(id string, limit int) ([]byte, error) {
	path, err := s.path(id)
	if err != nil {
		return nil, err
	}
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("image_expired")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil || len(b) > limit {
		return nil, errors.New("invalid_image")
	}
	return b, nil
}
func (s *PhotoStore) Delete(id string) error {
	path, err := s.path(id)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// A crash between saving a normalized image and committing its draft can leave
// an orphan. All objects are temporary; a one-hour margin avoids early expiry.
func (s *PhotoStore) DeleteExpiredObjects(now time.Time) error {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jpg") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".jpg")
		if _, err = s.path(id); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && now.Sub(info.ModTime()) >= 25*time.Hour {
			if err = s.Delete(id); err != nil {
				return err
			}
		}
	}
	return nil
}
