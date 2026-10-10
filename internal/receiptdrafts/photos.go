package receiptdrafts

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const MaxPhotoBytes = 8 << 20

var imageSlots = make(chan struct{}, 2)

func NormalizePhoto(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > MaxPhotoBytes {
		return nil, errors.New("invalid_image")
	}
	select {
	case imageSlots <- struct{}{}:
		defer func() { <-imageSlots }()
	default:
		return nil, errors.New("image_busy")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") || cfg.Width < 16 || cfg.Height < 16 || cfg.Width > 10000 || cfg.Height > 10000 || int64(cfg.Width)*int64(cfg.Height) > 10000000 {
		return nil, errors.New("invalid_image")
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, errors.New("invalid_image")
	}
	orientation := photoOrientation(raw, format)
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if orientation >= 5 {
		width, height = height, width
	}
	rotated := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			dx, dy := x, y
			switch orientation {
			case 2:
				dx = width - 1 - x
			case 3:
				dx, dy = width-1-x, height-1-y
			case 4:
				dy = height - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = width-1-y, x
			case 7:
				dx, dy = width-1-y, height-1-x
			case 8:
				dx, dy = y, height-1-x
			}
			rotated.Set(dx, dy, source.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	target := rotated
	if width > 2048 || height > 2048 {
		w, h := width, height
		if width >= height {
			w = 2048
			h = height * 2048 / width
		} else {
			h = 2048
			w = width * 2048 / height
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		target = image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(target, target.Bounds(), rotated, rotated.Bounds(), draw.Src, nil)
	}
	// Re-encoding removes EXIF, embedded thumbnails, appended content and all
	// original metadata. Only this private normalized image reaches vision.
	var out bytes.Buffer
	if err = jpeg.Encode(&out, target, &jpeg.Options{Quality: 90}); err != nil || out.Len() > MaxPhotoBytes {
		return nil, errors.New("invalid_image")
	}
	return out.Bytes(), nil
}
func tiffOrientation(b []byte) int {
	if len(b) < 8 {
		return 1
	}
	var order binary.ByteOrder
	if string(b[:2]) == "II" {
		order = binary.LittleEndian
	} else if string(b[:2]) == "MM" {
		order = binary.BigEndian
	} else {
		return 1
	}
	if order.Uint16(b[2:4]) != 42 {
		return 1
	}
	offset := int(order.Uint32(b[4:8]))
	if offset < 8 || offset > len(b)-2 {
		return 1
	}
	count := int(order.Uint16(b[offset : offset+2]))
	if count > 200 {
		return 1
	}
	for i := 0; i < count; i++ {
		at := offset + 2 + i*12
		if at > len(b)-12 {
			return 1
		}
		v := b[at : at+12]
		if order.Uint16(v[:2]) == 0x0112 && order.Uint16(v[2:4]) == 3 && order.Uint32(v[4:8]) == 1 {
			n := int(order.Uint16(v[8:10]))
			if n >= 1 && n <= 8 {
				return n
			}
			return 1
		}
	}
	return 1
}
func photoOrientation(raw []byte, format string) int {
	if format == "jpeg" {
		for at := 2; at+4 <= len(raw); {
			if raw[at] != 0xff {
				return 1
			}
			marker := raw[at+1]
			if marker == 0xda || marker == 0xd9 {
				return 1
			}
			n := int(binary.BigEndian.Uint16(raw[at+2 : at+4]))
			if n < 2 || n > len(raw)-(at+2) {
				return 1
			}
			if marker == 0xe1 && n >= 8 && string(raw[at+4:at+10]) == "Exif\x00\x00" {
				return tiffOrientation(raw[at+10 : at+2+n])
			}
			at += 2 + n
		}
	} else if format == "png" {
		for at := 8; at+12 <= len(raw); {
			n := int(binary.BigEndian.Uint32(raw[at : at+4]))
			if n < 0 || n > len(raw)-at-12 {
				return 1
			}
			if string(raw[at+4:at+8]) == "eXIf" {
				return tiffOrientation(raw[at+8 : at+8+n])
			}
			at += n + 12
		}
	} else if format == "webp" {
		for at := 12; at+8 <= len(raw); {
			n := int(binary.LittleEndian.Uint32(raw[at+4 : at+8]))
			if n < 0 || n > len(raw)-at-8 {
				return 1
			}
			if string(raw[at:at+4]) == "EXIF" {
				b := raw[at+8 : at+8+n]
				if len(b) >= 6 && string(b[:6]) == "Exif\x00\x00" {
					b = b[6:]
				}
				return tiffOrientation(b)
			}
			at += 8 + n + n%2
		}
	}
	return 1
}

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
func (s *PhotoStore) Save(data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxPhotoBytes {
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
func (s *PhotoStore) Read(id string) ([]byte, error) {
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
	b, err := io.ReadAll(io.LimitReader(f, MaxPhotoBytes+1))
	if err != nil || len(b) > MaxPhotoBytes {
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
