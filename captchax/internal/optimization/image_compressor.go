package optimization

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"sync"
)

type ImageCompressor struct {
	pool sync.Pool
}

func NewImageCompressor() *ImageCompressor {
	return &ImageCompressor{
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

func (c *ImageCompressor) CompressJPEG(img image.Image, quality int) ([]byte, error) {
	buf := c.pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		c.pool.Put(buf)
	}()

	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func (c *ImageCompressor) CompressPNG(img image.Image) ([]byte, error) {
	buf := c.pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		c.pool.Put(buf)
	}()

	err := png.Encode(buf, img)
	if err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func (c *ImageCompressor) CompressWebP(img image.Image, quality int) ([]byte, error) {
	buf := c.pool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		c.pool.Put(buf)
	}()

	if err := webpEncode(buf, img, quality); err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}
