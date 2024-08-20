package log

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"io"
	logv1 "loggy/api/v1"
	"os"
	"testing"
)

func TestLog(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T, log *Log){
		"append and read a record succeeds": testAppendRead,
		"offset out of range error":         testOutOfRangeErr,
		"init with existing segments":       testInitExisting,
		"reader":                            testReader,
		"truncate":                          testTruncate,
	} {
		t.Run(scenario, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "store-test")
			require.NoError(t, err)
			defer os.RemoveAll(dir)
			c := Config{}
			c.Segment.MaxStoreBytes = 32
			log, err := NewLog(dir, c)
			require.NoError(t, err)
			fn(t, log)
		})
	}
}

func testTruncate(t *testing.T, log *Log) {
	a := &logv1.Record{
		Value: []byte("hello world"),
	}
	for i := 0; i < 3; i++ {
		_, err := log.Append(a)
		require.NoError(t, err)
	}
	err := log.Truncate(1)
	require.NoError(t, err)

	_, err = log.Read(0)
	require.Error(t, err)
}

func testReader(t *testing.T, log *Log) {
	a := &logv1.Record{
		Value: []byte("hello world"),
	}
	off, err := log.Append(a)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	reader := log.Reader()
	b, err := io.ReadAll(reader)
	require.NoError(t, err)

	read := &logv1.Record{}
	err = proto.Unmarshal(b[lenWidth:], read)
	require.NoError(t, err)
	require.Equal(t, a.Value, read.Value)

}

func testInitExisting(t *testing.T, o *Log) {
	a := &logv1.Record{Value: []byte("hello world")}
	for i := 0; i < 3; i++ {
		_, err := o.Append(a)
		require.NoError(t, err)
	}
	require.NoError(t, o.Close())
	off, err := o.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	off, err = o.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)

	n, err := NewLog(o.Dir, o.Config)
	require.NoError(t, err)

	off, err = n.LowestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)
	off, err = n.HighestOffset()
	require.NoError(t, err)
	require.Equal(t, uint64(2), off)
}

func testOutOfRangeErr(t *testing.T, log *Log) {
	read, err := log.Read(1)
	require.Nil(t, read)
	var apiErr logv1.ErrOffsetOutOfRange
	errors.As(err, &apiErr)
	require.Equal(t, uint64(1), apiErr.Offset)
	require.Nil(t, read)
}

func testAppendRead(t *testing.T, log *Log) {
	a := &logv1.Record{Value: []byte("hello world")}
	off, err := log.Append(a)
	require.NoError(t, err)
	require.Equal(t, uint64(0), off)

	read, err := log.Read(off)
	require.NoError(t, err)
	require.Equal(t, a.Value, read.Value)
}
