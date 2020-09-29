package imgconv

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/sunshineplan/tiff"
)

const defaultOpacity = 128

var defaultFormat = formatOption{format: imaging.JPEG, encodeOption: []interface{}{imaging.JPEGQuality(75)}}

// Options represents options that can be used to configure a image operation.
type Options struct {
	Watermark *WatermarkOption
	Resize    *ResizeOption
	format    formatOption
}

// New return a default option.
func New() Options {
	return Options{format: defaultFormat}
}

// SetWatermark sets the value for the Watermark field.
func (o *Options) SetWatermark(mark string, opacity uint, random bool, offset image.Point) (*Options, error) {
	img, err := imaging.Open(mark)
	if err != nil {
		return nil, err
	}
	o.Watermark = &WatermarkOption{mark: mark, Mark: img, Random: random}
	if !random {
		o.Watermark.Offset = offset
	}
	if opacity == 0 {
		o.Watermark.Opacity = defaultOpacity
	} else {
		o.Watermark.Opacity = uint8(opacity)
	}
	return o, nil
}

// SetResize sets the value for the Resize field.
func (o *Options) SetResize(width, height int, percent float64) *Options {
	o.Resize = &ResizeOption{Width: width, Height: height, Percent: percent}
	return o
}

// SetFormat sets the value for the Format field.
func (o *Options) SetFormat(f string, option ...interface{}) error {
	var format imaging.Format
	var err error
	if format, err = imaging.FormatFromExtension(f); err != nil {
		return err
	}
	switch format {
	case imaging.TIFF:
		var opts []tiff.Options
		for _, i := range o.format.encodeOption {
			if opt, ok := i.(tiff.Options); ok {
				opts = append(opts, opt)
			}
		}
		var opt tiff.Options
		switch len(opts) {
		case 0:
			opt = tiff.Options{Compression: tiff.Deflate}
		case 1:
			opt = tiff.Options(opts[0])
		default:
			return errors.New("multiple TIFF compression option")
		}
		o.format = formatOption{format: format, encodeOption: []interface{}{opt}}
		return nil
	default:
		var opts []interface{}
		for _, i := range o.format.encodeOption {
			if opt, ok := i.(imaging.EncodeOption); ok {
				opts = append(opts, opt)
			}
		}
		o.format = formatOption{format: format, encodeOption: opts}
		return nil
	}
}

// Convert image by option
func (o *Options) Convert(src, dst string) error {
	output := o.format.path(dst)
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return os.ErrExist
	}

	var img image.Image
	var err error
	if ext := strings.ToLower(filepath.Ext(src)); ext == ".tif" || ext == ".tiff" {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		img, err = tiff.Decode(f)
	} else {
		img, err = imaging.Open(src)
	}
	if err != nil {
		return err
	}
	if o.Resize != nil {
		img = o.Resize.do(img)
	}
	if o.Watermark != nil {
		img = o.Watermark.do(img)
	}

	if reflect.DeepEqual(o.format, formatOption{}) {
		o.format = defaultFormat
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := o.format.save(img, output); err != nil {
		os.Remove(output)
		return err
	}

	return nil
}
