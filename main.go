package main

import (
	"encoding/base64"
	"github.com/sqweek/dialog"
	"hash/crc32"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"

	_ "image/png"
)

var palette = color.Palette{
	color.RGBA{0xFF, 0xFF, 0xFF, 0xFF},
	color.RGBA{0x00, 0x00, 0x00, 0xFF},
	color.RGBA{0x10, 0xAD, 0x42, 0xFF},
	color.RGBA{0xFF, 0xD6, 0x8C, 0xFF},
}

var vanillaLink = uint32(0xCA195F8A)
var vanillaBaby = uint32(0xEDA1184A)

func main() {
	fileDialog := dialog.File().Filter("Image Files (*.png)", "png")
	filename, err := fileDialog.Load()
	if err != nil {
		log.Fatal(err)
	}

	basename := filename[:len(filename)-len(filepath.Ext(filename))]

	reader, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}

	link := make([]byte, 0, 0x22E0)
	baby := make([]byte, 0, 0x0100)

	for y := 0; y < 288; y += 16 {
		for x := 0; x < 128; x += 8 {
			for y2 := 0; y2 < 16; y2 += 8 {
				if y == 272 && x >= 56 {
					continue
				}
				appendBlock(img, &link, x, y+y2)
			}
		}
	}
	modifiedLink := crc32.ChecksumIEEE(link) != vanillaLink
	if modifiedLink {
		os.WriteFile(basename+"_link.bin", link, 0644)
	}

	for y := 272; y < 288; y += 16 {
		for x := 64; x < 128; x += 8 {
			for y2 := 0; y2 < 16; y2 += 8 {
				appendBlock(img, &baby, x, y+y2)
			}
		}
	}
	modifiedBaby := crc32.ChecksumIEEE(baby) != vanillaBaby
	if modifiedBaby {
		os.WriteFile(basename+"_baby.bin", baby, 0644)
	}

	if !modifiedLink && !modifiedBaby {
		return
	}

	patchAges := []byte{0x50, 0x41, 0x54, 0x43, 0x48}
	patchSeasons := []byte{0x50, 0x41, 0x54, 0x43, 0x48}

	if modifiedLink {
		patchAges = append(patchAges, 0x06, 0x80, 0x00, 0x22, 0xE0)
		patchAges = append(patchAges, link...)
		patchSeasons = append(patchSeasons, 0x06, 0x80, 0x00, 0x22, 0xE0)
		patchSeasons = append(patchSeasons, link...)
	}
	if modifiedBaby {
		patchAges = append(patchAges, 0x06, 0xAC, 0xA0, 0x22, 0xE0)
		patchAges = append(patchAges, baby...)
		patchSeasons = append(patchSeasons, 0x06, 0xAC, 0x40, 0x22, 0xE0)
		patchSeasons = append(patchSeasons, baby...)
	}

	patchAges = append(patchAges, 0x45, 0x4F, 0x46)
	patchSeasons = append(patchSeasons, 0x45, 0x4F, 0x46)

	os.WriteFile(basename+"_ages.ips", patchAges, 0644)
	os.WriteFile(basename+"_seasons.ips", patchSeasons, 0644)

	yaml, err := os.Create(basename + ".yaml")
	if err != nil {
		log.Fatal(err)
	}
	defer yaml.Close()

	yaml.WriteString("common:\n")

	if modifiedLink {
		yaml.WriteString("  spr_link:\n")
		yaml.WriteString("    0x0: " + encode(link) + "\n")
	}
	if modifiedBaby {
		yaml.WriteString("  spr_link_baby:\n")
		yaml.WriteString("    0x0: " + encode(baby) + "\n")
	}
}

func appendBlock(img image.Image, bytes *[]byte, x, y int) {
	row := make([]int, 8)
	for j := 0; j < 8; j++ {
		for i := 0; i < 8; i++ {
			row[i] = palette.Index(img.At(x+i, y+j))
		}
		rowLow := 0
		rowHigh := 0
		for i := 0; i < 8; i++ {
			rowLow |= (row[i] & 1) << (7 - i)
			rowHigh |= ((row[i] & 2) >> 1) << (7 - i)
		}
		*bytes = append(*bytes, byte(rowLow), byte(rowHigh))
	}
}

func encode(data []byte) string {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(dst, data)
	return string(dst)
}
