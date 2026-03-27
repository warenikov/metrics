package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompress(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "empty input", input: []byte{}},
		{name: "plain text", input: []byte("hello world")},
		{name: "json payload", input: []byte(`{"id":"test","type":"gauge","value":1.5}`)},
		{name: "binary-like data", input: bytes.Repeat([]byte{0x00, 0xFF, 0xAB}, 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := Compress(tt.input)
			require.NoError(t, err)
			require.NotNil(t, buf)

			// проверяем что результат — валидный gzip
			gr, err := gzip.NewReader(buf)
			require.NoError(t, err)
			defer gr.Close()

			decoded, err := io.ReadAll(gr)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestNewReader(t *testing.T) {
	t.Run("valid gzip", func(t *testing.T) {
		original := []byte("test data for decompression")

		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write(original)
		require.NoError(t, err)
		require.NoError(t, gz.Close())

		gr, err := NewReader(&buf)
		require.NoError(t, err)
		defer gr.Close()

		decoded, err := io.ReadAll(gr)
		require.NoError(t, err)
		assert.Equal(t, original, decoded)
	})

	t.Run("invalid gzip returns error", func(t *testing.T) {
		gr, err := NewReader(strings.NewReader("not gzip data"))
		assert.Error(t, err)
		assert.Nil(t, gr)
	})
}

func TestCompressAndDecompress(t *testing.T) {
	original := []byte(`{"id":"cpu","type":"gauge","value":99.9}`)

	compressed, err := Compress(original)
	require.NoError(t, err)

	gr, err := NewReader(compressed)
	require.NoError(t, err)
	defer gr.Close()

	result, err := io.ReadAll(gr)
	require.NoError(t, err)
	assert.Equal(t, original, result)
}
