package postrender_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x-plugins/jx-secret-postrenderer/pkg/cmd/postrender"
	"github.com/stretchr/testify/require"
)

func TestPostrendererTransformIntegration(t *testing.T) {
	sourceFile := os.Getenv("JX_SECRET_INPUT")
	if sourceFile == "" {
		t.SkipNow()
		return
	}
	data, err := ioutil.ReadFile(sourceFile)
	require.NoError(t, err, "failed to read %s", sourceFile)

	_, o := postrender.NewCmdPostrender()

	result, err := o.Transform(string(data))
	require.NoError(t, err, "failed transform")

	t.Logf("created %s\n", result)
}
