package rgba

import (
	"fmt"
	"testing"
)

func TestPxColor(t *testing.T) {
	// testbytes := []byte("teststring")
	// expected := "85e63514d692e0136925bec920b4ac50c297c36774587c2ec2b86e8075001e73"

	// hash, err := DoubleSha256(testbytes)
	// if err != nil {
	// 	t.Error(err)
	// }
	// hash_str := fmt.Sprintf("%x", hash)
	// if hash_str != expected {
	// 	t.Errorf("expected %s but got %s", expected, hash)
	// }
	w := PxColor(2400)
	fmt.Printf("test got %v",w)
}
