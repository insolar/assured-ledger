package protokit

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaryProtoSize(t *testing.T) {
	require.Zero(t, BinaryProtoSize(0))
	require.Equal(t, 2, BinaryProtoSize(1))
}

func TestBinaryMarshalTo(t *testing.T) {
	var b [3]byte

	n, err := BinaryMarshalTo(b[:], false, func(b []byte) (int, error) {
		return 0, nil
	})
	require.NoError(t, err)
	require.Zero(t, n)

	n, err = BinaryMarshalTo(b[:], true, func(b []byte) (int, error) {
		return 0, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	n, err = BinaryMarshalTo(b[:], false, func(b []byte) (int, error) {
		b[0] = 99
		return 1, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, [...]byte{BinaryMarker, 99, 0}, b)

	n, err = BinaryMarshalTo(b[:], true, func(b []byte) (int, error) {
		b[0] = 99
		return 1, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, [...]byte{BinaryMarker, 99, 0}, b)

	n, err = BinaryMarshalTo(b[:], false, func(b []byte) (int, error) {
		return 1, errors.New("stub")
	})
	require.Error(t, err)
	require.Zero(t, n)
}

func TestBinaryMarshalToSizedBuffer(t *testing.T) {
	var b [3]byte

	n, err := BinaryMarshalToSizedBuffer(b[:], false, func(b []byte) (int, error) {
		return 0, nil
	})
	require.NoError(t, err)
	require.Zero(t, n)

	n, err = BinaryMarshalToSizedBuffer(b[:], true, func(b []byte) (int, error) {
		return 0, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	n, err = BinaryMarshalToSizedBuffer(b[:], false, func(b []byte) (int, error) {
		require.Equal(t, 2, len(b))
		b[1] = 99
		return 1, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, [...]byte{0, BinaryMarker, 99}, b)

	n, err = BinaryMarshalToSizedBuffer(b[:], true, func(b []byte) (int, error) {
		require.Equal(t, 2, len(b))
		b[1] = 99
		return 1, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, [...]byte{0, BinaryMarker, 99}, b)

	n, err = BinaryMarshalToSizedBuffer(b[:], false, func(b []byte) (int, error) {
		return 1, errors.New("stub")
	})
	require.Error(t, err)
	require.Zero(t, n)

	n, err = BinaryMarshalToSizedBuffer(b[:], false, func(b []byte) (int, error) {
		return 3, nil
	})
	require.Error(t, io.ErrShortBuffer)
	require.Zero(t, n)
}

func TestBinaryUnmarshal(t *testing.T) {
	err := BinaryUnmarshal(nil, func(b []byte) error {
		require.Nil(t, b)
		return nil
	})
	require.NoError(t, err)

	err = BinaryUnmarshal(make([]byte, 0, 1), func(b []byte) error {
		require.Nil(t, b)
		return nil
	})
	require.NoError(t, err)

	err = BinaryUnmarshal([]byte{BinaryMarker}, func(b []byte) error {
		require.NotNil(t, b)
		require.Zero(t, len(b))
		return nil
	})
	require.NoError(t, err)

	err = BinaryUnmarshal([]byte{BinaryMarker, 99, 0}, func(b []byte) error {
		require.Equal(t, 2, len(b))
		require.Equal(t, []byte{99, 0}, b)
		return nil
	})
	require.NoError(t, err)

	err = BinaryUnmarshal([]byte{BinaryMarker, 99, 0}, func(b []byte) error {
		require.Equal(t, 2, len(b))
		return errors.New("stub")
	})
	require.Error(t, err)

	err = BinaryUnmarshal([]byte{11, 99, 0}, func(b []byte) error {
		t.FailNow()
		return nil
	})
	require.Error(t, err)
}
