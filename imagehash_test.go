// Copyright 2017 The goimagehash Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goimagehash

import (
	"bufio"
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	"os"
	"reflect"
	"runtime"
	"testing"
)

func TestNewImageHash(t *testing.T) {
	for _, tt := range []struct {
		datas    [][]uint8
		hash1    Kind
		hash2    Kind
		distance int
		err      error
	}{
		{[][]uint8{{1, 0, 1, 1}, {0, 0, 0, 0}}, Unknown, Unknown, 3, nil},
		{[][]uint8{{0, 0, 0, 0}, {0, 0, 0, 0}}, Unknown, Unknown, 0, nil},
		{[][]uint8{{0, 0, 0, 0}, {0, 0, 0, 1}}, Unknown, Unknown, 1, nil},
		{[][]uint8{{0, 0, 0, 0}, {0, 0, 0, 1}}, Unknown, AHash, -1, errors.New("Image hashes's kind should be identical")},
	} {
		data1 := tt.datas[0]
		data2 := tt.datas[1]
		hash1 := NewImageHash(0, tt.hash1)
		hash2 := NewImageHash(0, tt.hash2)

		for i := 0; i < len(data1); i++ {
			if data1[i] == 1 {
				hash1.leftShiftSet(i)
			}
		}

		for i := 0; i < len(data2); i++ {
			if data2[i] == 1 {
				hash2.leftShiftSet(i)
			}
		}

		dis, err := hash1.Distance(hash2)
		if dis != tt.distance {
			t.Errorf("Distance between %v and %v expected as %d but got %d", data1, data2, tt.distance, dis)
		}
		if err != nil && err.Error() != tt.err.Error() {
			t.Errorf("Expected err %s, actual %s", tt.err, err)
		}
	}
}

func TestNil(t *testing.T) {
	hash := NewImageHash(0, AHash)
	dis, err := hash.Distance(nil)
	if err != errNoOther {
		t.Errorf("Expected err %s, actual %s", errNoOther, err)
	}
	if dis != -1 {
		t.Errorf("Distance is expected as %d but got %d", -1, dis)
	}
}

func TestSerialization(t *testing.T) {
	checkErr := func(err error) {
		if err != nil {
			t.Errorf("%v", err)
		}
	}

	methods := []func(img image.Image) (*ImageHash, error){
		AverageHash, PerceptionHash, DifferenceHash,
	}
	extMethods := []func(img image.Image, width int, height int) (*ExtImageHash, error){
		ExtAverageHash, ExtPerceptionHash, ExtDifferenceHash,
	}
	examples := []string{
		"_examples/sample1.jpg", "_examples/sample2.jpg", "_examples/sample3.jpg", "_examples/sample4.jpg",
	}

	for _, ex := range examples {
		file, err := os.Open(ex)
		checkErr(err)

		defer file.Close()

		img, _, err := image.Decode(file)
		checkErr(err)

		for _, method := range methods {
			methodStr := runtime.FuncForPC(reflect.ValueOf(method).Pointer()).Name()

			hash, err := method(img)
			checkErr(err)

			hex := hash.ToString()
			// len(kind) == 1, len(":") == 1, len(hash) == 16
			if len(hex) != 18 {
				t.Errorf("Got invalid hex string '%v'; %v of '%v'", hex, methodStr, ex)
			}

			reHash, err := ImageHashFromString(hex)
			checkErr(err)

			distance, err := hash.Distance(reHash)
			checkErr(err)

			if distance != 0 {
				t.Errorf("Original and unserialized objects should be identical, got distance=%v; %v of '%v'", distance, methodStr, ex)
			}
		}

		// test for ExtIExtImageHash
		for _, extMethod := range extMethods {
			extMethodStr := runtime.FuncForPC(reflect.ValueOf(extMethod).Pointer()).Name()
			sizeList := []int{8, 16}
			for _, size := range sizeList {
				hash, err := extMethod(img, size, size)
				checkErr(err)

				hex := hash.ToString()
				// len(kind) == 1, len(":") == 1
				if len(hex) != size*size/4+2 {
					t.Errorf("Got invalid hex string '%v'; %v of '%v'", hex, extMethodStr, ex)
				}

				reHash, err := ExtImageHashFromString(hex)
				checkErr(err)

				distance, err := hash.Distance(reHash)
				checkErr(err)

				if distance != 0 {
					t.Errorf("Original and unserialized objects should be identical, got distance=%v; %v of '%v'", distance, "ExtPerceptionHash", ex)
				}
			}
		}
	}

	// test for hashing empty string
	imageHash, err := ImageHashFromString("")
	if imageHash != nil {
		t.Errorf("Expected reHash to be nil, got %v", imageHash)
	}
	if err == nil {
		t.Errorf("Should got error for empty string")
	}
	extImageHash, err := ExtImageHashFromString("")
	if extImageHash != nil {
		t.Errorf("Expected reHash to be nil, got %v", extImageHash)
	}
	if err == nil {
		t.Errorf("Should got error for empty string")
	}

	// test for hashing invalid (non-hexadecimal) string
	extImageHash, err = ExtImageHashFromString("k:g")
}

func TestDifferentBitSizeHash(t *testing.T) {
	checkErr := func(err error) {
		if err != nil {
			t.Errorf("%v", err)
		}
	}
	file, err := os.Open("_examples/sample1.jpg")
	checkErr(err)
	defer file.Close()

	img, _, err := image.Decode(file)
	checkErr(err)

	hash1, _ := ExtAverageHash(img, 32, 32)
	hash2, _ := ExtDifferenceHash(img, 32, 32)
	_, err = hash1.Distance(hash2)
	if err == nil {
		t.Errorf("Should got error with different kinds of hashes")
	}
	hash3, _ := ExtAverageHash(img, 31, 31)
	_, err = hash1.Distance(hash3)
	if err == nil {
		t.Errorf("Should got error with different bits of hashes")
	}
}
func TestDumpAndLoad(t *testing.T) {
	checkErr := func(err error) {
		if err != nil {
			t.Errorf("%v", err)
		}
	}

	methods := []func(img image.Image) (*ImageHash, error){
		AverageHash, PerceptionHash, DifferenceHash,
	}
	examples := []string{
		"_examples/sample1.jpg", "_examples/sample2.jpg", "_examples/sample3.jpg", "_examples/sample4.jpg",
	}

	for _, ex := range examples {
		file, err := os.Open(ex)
		checkErr(err)

		defer file.Close()

		img, _, err := image.Decode(file)
		checkErr(err)

		for _, method := range methods {
			hash, err := method(img)
			checkErr(err)
			var b bytes.Buffer
			foo := bufio.NewWriter(&b)
			err = hash.Dump(foo)
			checkErr(err)
			foo.Flush()
			bar := bufio.NewReader(&b)
			reHash, err := LoadImageHash(bar)
			checkErr(err)

			distance, err := hash.Distance(reHash)
			checkErr(err)

			if distance != 0 {
				t.Errorf("Original and unserialized objects should be identical, got distance=%v", distance)
			}

			if hash.Bits() != 64 || reHash.Bits() != 64 {
				t.Errorf("Hash bits should be 64 but got, %v, %v", hash.Bits(), reHash.Bits())
			}
		}

		// test for ExtIExtImageHash
		extMethods := []func(img image.Image, width, height int) (*ExtImageHash, error){
			ExtAverageHash, ExtPerceptionHash, ExtDifferenceHash,
		}

		sizeList := []int{8, 16}
		for _, size := range sizeList {
			for _, method := range extMethods {
				hash, err := method(img, size, size)
				checkErr(err)
				var b bytes.Buffer
				foo := bufio.NewWriter(&b)
				err = hash.Dump(foo)
				checkErr(err)
				foo.Flush()
				bar := bufio.NewReader(&b)
				reHash, err := LoadExtImageHash(bar)
				checkErr(err)

				distance, err := hash.Distance(reHash)
				checkErr(err)

				if distance != 0 {
					t.Errorf("Original and unserialized objects should be identical, got distance=%v", distance)
				}

				if hash.Bits() != size*size || reHash.Bits() != size*size {
					t.Errorf("Hash bits should be 64 but got, %v, %v", hash.Bits(), reHash.Bits())
				}
			}
		}
	}

	// test for loading empty bytes buffer
	var b bytes.Buffer
	bar := bufio.NewReader(&b)
	_, err := LoadImageHash(bar)
	if err == nil {
		t.Errorf("Should got error for empty bytes buffer")
	}
	_, err = LoadExtImageHash(bar)
	if err == nil {
		t.Errorf("Should got error for empty bytes buffer")
	}
}

func TestImageHash(t *testing.T) {
	// Test cases
	testCases := []struct {
		name string
		kind Kind
		hash uint64
	}{
		{"Zero Hash", AHash, 0},
		{"Small Hash", AHash, 255},
		{"Large Hash", AHash, 1234567890123456789},
		{"Max Hash", PHash, 0xFFFFFFFFFFFFFFFF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Create ImageHash
			h := &ImageHash{
				kind: tc.kind,
				hash: tc.hash,
			}

			// Step 2: Convert to byte array
			kind, byteArr := h.ByteArr()
			if kind != tc.kind {
				t.Errorf("Expected kind %v, got %v", tc.kind, kind)
			}

			if len(byteArr) != 8 {
				t.Fatalf("Expected byte array length of 8, got %d", len(byteArr))
			}

			// Step 3: Create a new ImageHash and set values using FromByteArr
			newHash := &ImageHash{}
			err := newHash.FromByteArr(tc.kind, byteArr)
			if err != nil {
				t.Fatalf("FromByteArr returned error: %v", err)
			}

			// Step 4: Verify kind and hash
			if newHash.kind != tc.kind {
				t.Errorf("FromByteArr kind mismatch: expected %v, got %v", tc.kind, newHash.kind)
			}
			if newHash.hash != tc.hash {
				t.Errorf("FromByteArr hash mismatch: expected %v, got %v", tc.hash, newHash.hash)
			}

			// Step 5: Compare byte arrays
			_, newByteArr := newHash.ByteArr()
			if !bytes.Equal(byteArr, newByteArr) {
				t.Errorf("Byte arrays do not match: original %v, new %v", byteArr, newByteArr)
			}
		})
	}
}

func TestFromByteArrError(t *testing.T) {
	h := &ImageHash{}

	// Invalid byte array size
	err := h.FromByteArr(AHash, []byte{0x00, 0x01})
	if err == nil {
		t.Error("Expected error for incorrect byte array size, but got none")
	}
}
