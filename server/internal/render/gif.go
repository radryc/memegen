package render

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"os"
	"runtime"
	"sync"

	"github.com/soniakeys/quant/median"
	xdraw "golang.org/x/image/draw"
)

const (
	maxGIFSide   = 900
	maxGIFFrames = 150
)

// RenderGIFFile composites meme captions onto every frame of an animated GIF.
func RenderGIFFile(path, topText, bottomText, rawStyle string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := gif.DecodeAll(f)
	if err != nil {
		return nil, fmt.Errorf("decode gif template: %w", err)
	}
	return RenderGIF(src, topText, bottomText, ParseStyle(rawStyle))
}

// RenderGIF renders captions over a decoded GIF, preserving frame delays and
// loop count. Frames are composed with their disposal modes, then quantized.
func RenderGIF(src *gif.GIF, topText, bottomText string, style Style) ([]byte, error) {
	if len(src.Image) == 0 {
		return nil, fmt.Errorf("gif has no frames")
	}
	if len(src.Image) > maxGIFFrames {
		return nil, fmt.Errorf("gif has too many frames (max %d)", maxGIFFrames)
	}
	bounds := src.Image[0].Bounds()
	for _, frame := range src.Image {
		bounds = bounds.Union(frame.Bounds())
	}
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || width*height > 25_000_000 {
		return nil, fmt.Errorf("gif dimensions are not supported")
	}
	scale := 1.0
	if longest := max(width, height); longest > maxGIFSide {
		scale = float64(maxGIFSide) / float64(longest)
	}
	outW := max(1, int(float64(width)*scale))
	outH := max(1, int(float64(height)*scale))

	full := image.NewRGBA(image.Rect(0, 0, width, height))
	frames := make([]*image.Paletted, len(src.Image))
	delays := make([]int, len(src.Image))

	// Caption drawing and colour quantization are the expensive parts. Compose
	// frames in order (disposal is order-dependent) and hand each finished
	// frame to a small worker pool.
	type frameJob struct {
		index int
		img   *image.RGBA
	}
	type frameResult struct {
		index int
		frame *image.Paletted
		err   error
	}
	workers := min(4, runtime.NumCPU())
	jobs := make(chan frameJob, workers)
	results := make(chan frameResult, len(src.Image))
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if err := drawCaptions(job.img, topText, bottomText, style); err != nil {
					results <- frameResult{index: job.index, err: err}
					continue
				}
				pal := median.Quantizer(256).Palette(job.img).ColorPalette()
				paletted := image.NewPaletted(job.img.Bounds(), pal)
				draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), job.img, job.img.Bounds().Min)
				results <- frameResult{index: job.index, frame: paletted}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var previous *image.RGBA
	prevBounds := image.Rectangle{}
	prevDisposal := byte(0)

	for i, frame := range src.Image {
		if i > 0 {
			switch prevDisposal {
			case gif.DisposalBackground:
				draw.Draw(full, prevBounds, image.Transparent, image.Point{}, draw.Src)
			case gif.DisposalPrevious:
				if previous != nil {
					copy(full.Pix, previous.Pix)
				}
			}
		}
		disposal := byte(0)
		if i < len(src.Disposal) {
			disposal = src.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			previous = image.NewRGBA(full.Bounds())
			copy(previous.Pix, full.Pix)
		}
		draw.Draw(full, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		outFrame := image.NewRGBA(image.Rect(0, 0, outW, outH))
		if outW == width && outH == height {
			draw.Draw(outFrame, outFrame.Bounds(), full, image.Point{}, draw.Src)
		} else {
			xdraw.CatmullRom.Scale(outFrame, outFrame.Bounds(), full, full.Bounds(), draw.Src, nil)
		}
		jobs <- frameJob{index: i, img: outFrame}

		delay := 10
		if i < len(src.Delay) && src.Delay[i] > 0 {
			delay = src.Delay[i]
		}
		delays[i] = delay

		prevBounds = frame.Bounds()
		prevDisposal = disposal
	}
	close(jobs)

	var firstErr error
	for result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		frames[result.index] = result.frame
	}
	if firstErr != nil {
		return nil, firstErr
	}

	var buf bytes.Buffer
	out := &gif.GIF{
		Image:     frames,
		Delay:     delays,
		LoopCount: src.LoopCount,
		Config: image.Config{
			ColorModel: frames[0].Palette,
			Width:      outW,
			Height:     outH,
		},
	}
	if err := gif.EncodeAll(&buf, out); err != nil {
		return nil, fmt.Errorf("encode gif: %w", err)
	}
	return buf.Bytes(), nil
}
