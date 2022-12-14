package server

import (
	_ "embed"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDecryptor(t *testing.T) {
	tests := []struct {
		name      string
		keyFile   string
		wantError bool
		errorText string
	}{
		{
			name:      "TestOne",
			keyFile:   "testkey.priv",
			wantError: false,
			errorText: "",
		},
		{
			name:      "TestTwo",
			keyFile:   "testkeyDontExist.priv",
			wantError: true,
			errorText: "open testkeyDontExist.priv: no such file or directory",
		},
		{
			name:      "TestThree",
			keyFile:   "testbadkey.priv",
			wantError: true,
			errorText: "asn1: syntax error: data truncated",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDecryptor(tt.keyFile)
			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorText, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecrypt(t *testing.T) {

	tests := []struct {
		name      string
		keyFile   string
		wantError bool
		encrypted []byte
		decrypted []byte
	}{
		{
			name:      "TestOne",
			keyFile:   "testkey.priv",
			wantError: false,
			encrypted: []byte{150, 175, 105, 166, 14, 50, 47, 198, 72, 24, 248, 111, 191, 191, 46, 169, 41, 70, 123, 188, 39, 139, 57, 2, 35, 98, 58, 68, 55, 72, 191, 114, 227, 224, 221, 199, 213, 179, 56, 240, 246, 54, 27, 176, 15, 135, 61, 171, 49, 104, 156, 40, 101, 174, 193, 58, 14, 16, 196, 173, 9, 55, 209, 129, 37, 206, 248, 171, 75, 33, 39, 157, 152, 12, 48, 94, 160, 202, 194, 32, 65, 96, 255, 245, 190, 29, 98, 91, 53, 255, 254, 218, 96, 47, 173, 72, 79, 56, 61, 156, 130, 166, 101, 45, 10, 128, 254, 12, 183, 156, 157, 27, 98, 146, 19, 156, 34, 109, 54, 70, 206, 250, 65, 3, 32, 178, 239, 163, 73, 80, 13, 105, 173, 73, 56, 238, 94, 84, 43, 193, 71, 10, 75, 51, 92, 18, 180, 88, 178, 82, 115, 77, 207, 111, 255, 34, 213, 226, 87, 24, 238, 2, 215, 125, 24, 35, 29, 179, 187, 3, 181, 234, 86, 130, 26, 172, 6, 155, 141, 145, 224, 82, 15, 165, 226, 226, 6, 114, 37, 97, 188, 165, 21, 149, 155, 12, 7, 183, 89, 103, 78, 184, 241, 184, 53, 134, 1, 44, 182, 220, 50, 6, 157, 79, 75, 51, 167, 96, 244, 252, 101, 62, 218, 125, 138, 65, 190, 167, 128, 52, 243, 249, 158, 45, 70, 101, 147, 86, 0, 96, 96, 182, 247, 42, 238, 69, 175, 88, 219, 152, 19, 137, 158, 39, 107, 102, 97, 246, 14, 28, 18, 115, 118, 116, 166, 132, 0, 81, 189, 226, 249, 133, 148, 155, 32, 213, 132, 37, 157, 12, 214, 251, 132, 62, 210, 223, 1, 123, 219, 79, 65, 208, 147, 150, 240, 20, 197, 159, 240, 98, 0, 101, 134, 58, 175, 164, 78, 248, 67, 135, 218, 27, 55, 214, 112, 109, 133, 9, 157, 243, 175, 3, 103, 141, 17, 194, 126, 17, 26, 41, 55, 70, 140, 113, 126, 188, 161, 21, 57, 248, 150, 233, 37, 118, 90, 19, 194, 56, 180, 248, 125, 90, 133, 180, 14, 22, 138, 209, 104, 50, 137, 21, 210, 121, 31, 14, 4, 126, 203, 126, 36, 242, 152, 128, 114, 81, 192, 138, 193, 178, 26, 47, 3, 119, 120, 206, 75, 114, 12, 216, 112, 87, 150, 134, 111, 234, 46, 185, 215, 197, 110, 28, 100, 224, 230, 37, 130, 57, 159, 255, 181, 173, 32, 104, 229, 102, 246, 46, 213, 98, 36, 142, 159, 169, 36, 9, 224, 54, 168, 150, 177, 136, 74, 42, 5, 106, 0, 213, 147, 209, 163, 206, 154, 9, 165, 104, 219, 136, 61, 72, 13, 84, 12, 54, 214, 169, 246, 230, 156, 72, 90, 225, 221, 133, 236, 11, 48, 197, 203, 191, 71, 220, 97, 232, 248, 44, 33, 252, 141, 163, 190, 53, 42, 121, 132, 68, 248, 112, 86, 209, 249, 100, 196, 29, 164, 121, 189, 25, 3, 143, 253, 43, 90, 251, 236, 179, 24, 230, 133, 164, 200, 250},
			decrypted: []byte{123, 34, 105, 100, 34, 58, 34, 67, 80, 85, 117, 116, 105, 108, 105, 122, 97, 116, 105, 111, 110, 49, 49, 34, 44, 34, 116, 121, 112, 101, 34, 58, 34, 103, 97, 117, 103, 101, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 50, 46, 52, 48, 50, 52, 48, 50, 52, 48, 50, 51, 56, 52, 50, 52, 57, 44, 34, 104, 97, 115, 104, 34, 58, 34, 101, 48, 51, 51, 49, 55, 55, 51, 98, 101, 98, 56, 100, 97, 97, 51, 57, 53, 57, 99, 56, 54, 51, 101, 53, 50, 100, 99, 52, 53, 97, 57, 99, 54, 49, 57, 101, 101, 50, 52, 49, 100, 97, 54, 55, 54, 55, 100, 101, 54, 57, 102, 48, 101, 51, 52, 101, 55, 99, 48, 50, 55, 51, 102, 34, 125},
		},
		{
			name:      "TestTwo",
			keyFile:   "testkey.priv",
			wantError: true,
			encrypted: []byte{166, 14, 50, 47, 198, 72, 24, 248, 111, 191, 191, 46, 169, 41, 70, 123, 188, 39, 139, 57, 2, 35, 98, 58, 68, 55, 72, 191, 114, 227, 224, 221, 199, 213, 179, 56, 240, 246, 54, 27, 176, 15, 135, 61, 171, 49, 104, 156, 40, 101, 174, 193, 58, 14, 16, 196, 173, 9, 55, 209, 129, 37, 206, 248, 171, 75, 33, 39, 157, 152, 12, 48, 94, 160, 202, 194, 32, 65, 96, 255, 245, 190, 29, 98, 91, 53, 255, 254, 218, 96, 47, 173, 72, 79, 56, 61, 156, 130, 166, 101, 45, 10, 128, 254, 12, 183, 156, 157, 27, 98, 146, 19, 156, 34, 109, 54, 70, 206, 250, 65, 3, 32, 178, 239, 163, 73, 80, 13, 105, 173, 73, 56, 238, 94, 84, 43, 193, 71, 10, 75, 51, 92, 18, 180, 88, 178, 82, 115, 77, 207, 111, 255, 34, 213, 226, 87, 24, 238, 2, 215, 125, 24, 35, 29, 179, 187, 3, 181, 234, 86, 130, 26, 172, 6, 155, 141, 145, 224, 82, 15, 165, 226, 226, 6, 114, 37, 97, 188, 165, 21, 149, 155, 12, 7, 183, 89, 103, 78, 184, 241, 184, 53, 134, 1, 44, 182, 220, 50, 6, 157, 79, 75, 51, 167, 96, 244, 252, 101, 62, 218, 125, 138, 65, 190, 167, 128, 52, 243, 249, 158, 45, 70, 101, 147, 86, 0, 96, 96, 182, 247, 42, 238, 69, 175, 88, 219, 152, 19, 137, 158, 39, 107, 102, 97, 246, 14, 28, 18, 115, 118, 116, 166, 132, 0, 81, 189, 226, 249, 133, 148, 155, 32, 213, 132, 37, 157, 12, 214, 251, 132, 62, 210, 223, 1, 123, 219, 79, 65, 208, 147, 150, 240, 20, 197, 159, 240, 98, 0, 101, 134, 58, 175, 164, 78, 248, 67, 135, 218, 27, 55, 214, 112, 109, 133, 9, 157, 243, 175, 3, 103, 141, 17, 194, 126, 17, 26, 41, 55, 70, 140, 113, 126, 188, 161, 21, 57, 248, 150, 233, 37, 118, 90, 19, 194, 56, 180, 248, 125, 90, 133, 180, 14, 22, 138, 209, 104, 50, 137, 21, 210, 121, 31, 14, 4, 126, 203, 126, 36, 242, 152, 128, 114, 81, 192, 138, 193, 178, 26, 47, 3, 119, 120, 206, 75, 114, 12, 216, 112, 87, 150, 134, 111, 234, 46, 185, 215, 197, 110, 28, 100, 224, 230, 37, 130, 57, 159, 255, 181, 173, 32, 104, 229, 102, 246, 46, 213, 98, 36, 142, 159, 169, 36, 9, 224, 54, 168, 150, 177, 136, 74, 42, 5, 106, 0, 213, 147, 209, 163, 206, 154, 9, 165, 104, 219, 136, 61, 72, 13, 84, 12, 54, 214, 169, 246, 230, 156, 72, 90, 225, 221, 133, 236, 11, 48, 197, 203, 191, 71, 220, 97, 232, 248, 44, 33, 252, 141, 163, 190, 53, 42, 121, 132, 68, 248, 112, 86, 209, 249, 100, 196, 29, 164, 121, 189, 25, 3, 143, 253, 43, 90, 251, 236, 179, 24, 230, 133, 164, 200, 250},
			decrypted: []byte{123, 34, 105, 100, 34, 58, 34, 67, 80, 85, 117, 116, 105, 108, 105, 122, 97, 116, 105, 111, 110, 49, 49, 34, 44, 34, 116, 121, 112, 101, 34, 58, 34, 103, 97, 117, 103, 101, 34, 44, 34, 118, 97, 108, 117, 101, 34, 58, 50, 46, 52, 48, 50, 52, 48, 50, 52, 48, 50, 51, 56, 52, 50, 52, 57, 44, 34, 104, 97, 115, 104, 34, 58, 34, 101, 48, 51, 51, 49, 55, 55, 51, 98, 101, 98, 56, 100, 97, 97, 51, 57, 53, 57, 99, 56, 54, 51, 101, 53, 50, 100, 99, 52, 53, 97, 57, 99, 54, 49, 57, 101, 101, 50, 52, 49, 100, 97, 54, 55, 54, 55, 100, 101, 54, 57, 102, 48, 101, 51, 52, 101, 55, 99, 48, 50, 55, 51, 102, 34, 125},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDecryptor(tt.keyFile)
			if err != nil {
				log.Fatal(err)
			}

			result, err := d.decrypt(tt.encrypted)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.decrypted, result)
			}

		})
	}
}
