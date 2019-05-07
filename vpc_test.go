package awsomvpc

import (
	"github.com/hekonsek/random-strings"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateVpc(t *testing.T) {
	t.Parallel()

	// Given
	name := randomstrings.ForHumanWithHash()
	defer func() {
		err := DeleteVpc(name)
		assert.NoError(t, err)
	}()

	// When
	err := NewVpcBuilder(name).Create()
	assert.NoError(t, err)

	// Then
	exists, err := VpcExistsByName(name)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateVpcWithThreeSubnets(t *testing.T) {
	t.Parallel()

	// Given
	name := randomstrings.ForHumanWithHash()
	defer func() {
		err := DeleteVpc(name)
		assert.NoError(t, err)
	}()

	// When
	err := NewVpcBuilder(name).Create()
	assert.NoError(t, err)

	// Then
	subnets, err := VpcSubnetsByName(name)
	assert.NoError(t, err)
	assert.Len(t, subnets, 3)
}
