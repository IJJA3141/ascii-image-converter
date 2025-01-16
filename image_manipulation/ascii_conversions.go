/*
Copyright © 2021 Zoraiz Hassan <hzoraiz8@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package image_conversions

import (
	"math"
)

var (
	// Reference taken from http://paulbourke.net/dataformats/asciiart/
	asciiTableSimple   = " .:-=+*#%@"
	asciiTableDetailed = " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"

	// Structure for braille dots
	BrailleStruct = [4][2]int{
		{0x1, 0x8},
		{0x2, 0x10},
		{0x4, 0x20},
		{0x40, 0x80},
	}

	BrailleThreshold uint32
)

// For each individual element of imgSet in ConvertToASCIISlice()
const MAX_VAL float64 = 255

type AsciiChar struct {
	OriginalColor string
	SetColor      string
	Simple        string
	RgbValue      [3]uint32
}

/*
Converts the 2D image_conversions.AsciiPixel slice of image data (each instance representing each compressed pixel of original image)
to a 2D image_conversions.AsciiChar slice

If complex parameter is true, values are compared to 70 levels of color density in ASCII characters.
Otherwise, values are compared to 10 levels of color density in ASCII characters.
*/
func ConvertToAsciiChars(imgSet [][]AsciiPixel, negative, colored, grayscale, complex, colorBg bool, customMap string, fontColor [3]int) ([][]AsciiChar, error) {

	height := len(imgSet)
	width := len(imgSet[0])

	chosenTable := map[int]string{}

	// Turn ascii character-set string into map[int]string{} literal
	if customMap == "" {
		var charSet string

		if complex {
			charSet = asciiTableDetailed
		} else {
			charSet = asciiTableSimple
		}

		for index, char := range charSet {
			chosenTable[index] = string(char)
		}

	} else {
		chosenTable = map[int]string{}

		for index, char := range customMap {
			chosenTable[index] = string(char)
		}
	}

	var result [][]AsciiChar

	for i := 0; i < height; i++ {

		var tempSlice []AsciiChar

		for j := 0; j < width; j++ {
			value := float64(imgSet[i][j].charDepth)

			// Gets appropriate string index from chosenTable by percentage comparisons with its length
			tempFloat := (value / MAX_VAL) * float64(len(chosenTable))
			if value == MAX_VAL {
				tempFloat = float64(len(chosenTable) - 1)
			}
			tempInt := int(tempFloat)

			var r, g, b int

			if colored {
				r = int(imgSet[i][j].rgbValue[0])
				g = int(imgSet[i][j].rgbValue[1])
				b = int(imgSet[i][j].rgbValue[2])
			} else {
				r = int(imgSet[i][j].grayscaleValue[0])
				g = int(imgSet[i][j].grayscaleValue[1])
				b = int(imgSet[i][j].grayscaleValue[2])
			}

			if negative {
				// Select character from opposite side of table as well as turn pixels negative
				r = 255 - r
				g = 255 - g
				b = 255 - b

				// To preserve negative rgb values for saving png image later down the line, since it uses imgSet
				if colored {
					imgSet[i][j].rgbValue = [3]uint32{uint32(r), uint32(g), uint32(b)}
				} else {
					imgSet[i][j].grayscaleValue = [3]uint32{uint32(r), uint32(g), uint32(b)}
				}

				tempInt = (len(chosenTable) - 1) - tempInt
			}

			var char AsciiChar

			asciiChar := chosenTable[tempInt]
			char.Simple = asciiChar

			var err error
			if colorBg {
				char.OriginalColor, err = getColoredCharForTerm(uint8(r), uint8(g), uint8(b), asciiChar, true)
			} else {
				char.OriginalColor, err = getColoredCharForTerm(uint8(r), uint8(g), uint8(b), asciiChar, false)
			}
			if (colored || grayscale) && err != nil {
				return nil, err
			}

			// If font color is not set, use a simple string. Otherwise, use True color
			if fontColor != [3]int{255, 255, 255} {
				fcR := fontColor[0]
				fcG := fontColor[1]
				fcB := fontColor[2]

				if colorBg {
					char.SetColor, err = getColoredCharForTerm(uint8(fcR), uint8(fcG), uint8(fcB), asciiChar, true)
				} else {
					char.SetColor, err = getColoredCharForTerm(uint8(fcR), uint8(fcG), uint8(fcB), asciiChar, false)
				}
				if err != nil {
					return nil, err
				}
			}

			if colored {
				char.RgbValue = imgSet[i][j].rgbValue
			} else {
				char.RgbValue = imgSet[i][j].grayscaleValue
			}

			tempSlice = append(tempSlice, char)
		}
		result = append(result, tempSlice)
	}

	return result, nil
}

/*
Converts the 2D image_conversions.AsciiPixel slice of image data (each instance representing each compressed pixel of original image)
to a 2D image_conversions.AsciiChar slice

Unlike ConvertToAsciiChars(), this function calculates braille characters instead of ascii
*/
func ConvertToBrailleChars(imgSet [][]AsciiPixel, negative, colored, grayscale, colorBg bool, fontColor [3]int, threshold int) ([][]AsciiChar, error) {

	BrailleThreshold = uint32(threshold)

	height := len(imgSet)
	width := len(imgSet[0])

	var result [][]AsciiChar

	for i := 0; i < height; i += 4 {

		var tempSlice []AsciiChar

		for j := 0; j < width; j += 2 {

			brailleChar := getBrailleChar(i, j, negative, imgSet)

			var r, g, b int

			if colored {
				r = int(imgSet[i][j].rgbValue[0])
				g = int(imgSet[i][j].rgbValue[1])
				b = int(imgSet[i][j].rgbValue[2])
			} else {
				r = int(imgSet[i][j].grayscaleValue[0])
				g = int(imgSet[i][j].grayscaleValue[1])
				b = int(imgSet[i][j].grayscaleValue[2])
			}

			if negative {
				// Select character from opposite side of table as well as turn pixels negative
				r = 255 - r
				g = 255 - g
				b = 255 - b

				if colored {
					imgSet[i][j].rgbValue = [3]uint32{uint32(r), uint32(g), uint32(b)}
				} else {
					imgSet[i][j].grayscaleValue = [3]uint32{uint32(r), uint32(g), uint32(b)}
				}
			}

			var char AsciiChar

			char.Simple = brailleChar

			var err error
			if colorBg {
				char.OriginalColor, err = getColoredCharForTerm(uint8(r), uint8(g), uint8(b), brailleChar, true)
			} else {
				char.OriginalColor, err = getColoredCharForTerm(uint8(r), uint8(g), uint8(b), brailleChar, false)
			}
			if (colored || grayscale) && err != nil {
				return nil, err
			}

			// If font color is not set, use a simple string. Otherwise, use True color
			if fontColor != [3]int{255, 255, 255} {
				fcR := fontColor[0]
				fcG := fontColor[1]
				fcB := fontColor[2]

				if colorBg {
					char.SetColor, err = getColoredCharForTerm(uint8(fcR), uint8(fcG), uint8(fcB), brailleChar, true)
				} else {
					char.SetColor, err = getColoredCharForTerm(uint8(fcR), uint8(fcG), uint8(fcB), brailleChar, false)
				}
				if err != nil {
					return nil, err
				}
			}

			if colored {
				char.RgbValue = imgSet[i][j].rgbValue
			} else {
				char.RgbValue = imgSet[i][j].grayscaleValue
			}

			tempSlice = append(tempSlice, char)
		}

		result = append(result, tempSlice)
	}

	return result, nil
}

// Iterate through the BrailleStruct table to see which dots need to be highlighted
func getBrailleChar(x, y int, negative bool, imgSet [][]AsciiPixel) string {

	brailleChar := 0x2800

	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			if negative {
				if imgSet[x+i][y+j].charDepth <= BrailleThreshold {
					brailleChar += BrailleStruct[i][j]
				}
			} else {
				if imgSet[x+i][y+j].charDepth >= BrailleThreshold {
					brailleChar += BrailleStruct[i][j]
				}
			}
		}
	}

	return string(brailleChar)
}

func Convolution(matrix [][]int, kernel [][]int) [][]int {

	result := make([][]int, len(matrix))
	rj := len(kernel[0]) / 2
	ri := len(kernel) / 2

	for i := range result {
		result[i] = make([]int, len(matrix[0]))
	}

	for m, col := range result {
		for n := range col {

			for i, kerCol := range kernel {

				x := m - ri + i

				if x < 0 {
					x = 0
				} else {
					if x >= len(result) {
						x = len(result) - 1
					}
				}

				for j := range kerCol {

					y := n - rj + j

					if y < 0 {
						y = 0
					} else {
						if y >= len(col) {
							y = len(col) - 1
						}
					}

					result[m][n] += matrix[x][y] * kernel[i][j]
				}
			}
		}
	}

	return result
}

type PixelAngle struct {
	Angle    float64
	Gradiant int
	Char     string
	X        int
	Y        int
}

// apply the Sobel filter to a grayscale matrix
// returns a matrix of angles at each pixel
func SobelFilter(grayscale [][]int, threshold float64) []PixelAngle {

	gX := Convolution(grayscale, [][]int{
		{1, 0, -1},
		{2, 0, -2},
		{1, 0, -1},
	})

	gY := Convolution(grayscale, [][]int{
		{1, 2, 1},
		{0, 0, 0},
		{-1, -2, -1},
	})

	var angles []PixelAngle

	for m, col := range grayscale {
		for n := range col {

			gradiant := math.Sqrt(float64(gY[m][n] ^ 2 + gX[m][n] ^ 2))

			if gradiant < threshold {
				continue
			}

			var pixel PixelAngle

			pixel.Gradiant = int(gradiant)

			pixel.Angle = math.Atan2(float64(gY[m][n]), float64(gX[m][n]))
			pixel.Char = AngleToAscii(pixel.Angle)
			pixel.X = m
			pixel.Y = n

			angles = append(angles, pixel)
		}
	}

	return angles
}

// takes angle from -π and π
// and returns the corresponding ascii angle
func AngleToAscii(angle float64) string {
	if angle < -7*math.Pi/8 {
		return "|"
	}

	if angle < -5*math.Pi/8 {
		return "/"
	}

	if angle < -3*math.Pi/8 {
		return "-"
	}

	if angle < -math.Pi/8 {
		return "\\"
	}

	if angle < math.Pi/8 {
		return "|"
	}

	if angle < 3*math.Pi/8 {
		return "/"
	}

	if angle < 5*math.Pi/8 {
		return "-"
	}

	if angle < 7*math.Pi/8 {
		return "\\"
	}

	return "|"
}

func AddEdgeDetection(asciiSet [][]AsciiChar, imgSet [][]AsciiPixel, threshold float64) [][]AsciiChar {
	var (
		// https://en.wikipedia.org/wiki/Grayscale
		cr = 0.2126
		cb = 0.7152
		cg = 0.0722
	)

	grayscale := make([][]int, len(imgSet))

	for i, col := range imgSet {
		grayscale[i] = make([]int, len(col))

		for j := range imgSet {
			grayscale[i][j] = int(float64(imgSet[i][j].grayscaleValue[0])*cr + float64(imgSet[i][j].grayscaleValue[1])*cb + float64(imgSet[i][j].grayscaleValue[2])*cg)
		}
	}

	edges := SobelFilter(grayscale, threshold)

	for _, edge := range edges {
		asciiSet[edge.X][edge.Y].Simple = edge.Char
	}

	return asciiSet
}
