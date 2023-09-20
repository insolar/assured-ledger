package refmap

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMappedRefMap_LoadBucket(t *testing.T) {
	require.NotNil(t, emptyBucketMarker)

	//require.LessOrEqual(t, reference.LocalBinarySize, int(bucketKeyType.Size()))
	//require.LessOrEqual(t, reference.LocalBinarySize, bucketKeySize)
	//vf, _ := bucketKeyL1type.FieldByName("value")
	//require.Equal(t, reference.LocalBinarySize, int(vf.Offset))
	tp := reflect.TypeOf(mappedBucket{})
	fmt.Println(int(tp.Size()), tp.Align(), tp.FieldAlign())
	for i := 0; i < tp.NumField(); i++ {
		f := tp.Field(i)
		fmt.Println(f.Offset, f.Name, int(f.Type.Size()), f.Type.Align())
	}
}
