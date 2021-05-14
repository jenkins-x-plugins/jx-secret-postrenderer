package postrender

import (
	"context"
	"fmt"
	"github.com/jenkins-x-plugins/jx-secret/pkg/extsecrets/secretfacade"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/convert"
	"github.com/jenkins-x-plugins/jx-secret/pkg/cmd/populate"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	yaml2 "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"

	"github.com/jenkins-x-plugins/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/spf13/cobra"
)

var (
	// resourcesSeparator is used to separate multiple objects stored in the same YAML file
	resourcesSeparator = "---\n"

	cmdLong = templates.LongDesc(`
		A helm postrender to convert any Secret resources into ExternalSecret resources
`)

	cmdExample = templates.Examples(`
		# lets post render
		helm install --postrender 'jx secret postrender'  myname mychart
	`)

	secretFilter = kyamls.ParseKindFilter("v1/Secret")
)

// Options the options for the command
type Options struct {
	EnvOptions

	ConvertOptions  convert.Options
	PopulateOptions populate.Options

	SecretCount int
}

type EnvOptions struct {
	options.BaseOptions

	VaultMountPoint  string `env:"JX_VAULT_MOUNT_POINT"`
	VaultRole        string `env:"JX_VAULT_ROLE"`
	Dir              string `env:"JX_DIR"`
	DefaultNamespace string `env:"JX_DEFAULT_NAMESPACE"`

	// DisablePopulate disables if external secrets are populated from the helm secret data if not populated already
	DisablePopulate bool `env:"JX_NO_POPULATE"`
}

// NewCmdPostrender creates a command object for the command
func NewCmdPostrender() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "postrender",
		Short:   "A helm postrender to convert any Secret resources into ExternalSecret resources",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			if err != nil {
				os.Exit(1)
			}
		},
	}
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return errors.Wrapf(err, "failed to read standard input")
	}
	result, err := o.Transform(string(data))

	fmt.Println(result)
	return err
}

// Transform transforms the results
func (o *Options) Transform(text string) (string, error) {
	ctx := context.TODO()
	err := envconfig.Process(ctx, &o.EnvOptions)
	if err != nil {
		return "", errors.Wrapf(err, "failed to process environment options")
	}

	o.ConvertOptions.BaseOptions = o.EnvOptions.BaseOptions
	o.ConvertOptions.BatchMode = true
	o.ConvertOptions.DefaultNamespace = o.EnvOptions.DefaultNamespace
	o.ConvertOptions.Dir = o.EnvOptions.Dir
	o.ConvertOptions.VaultMountPoint = o.EnvOptions.VaultMountPoint
	o.ConvertOptions.VaultRole = o.EnvOptions.VaultRole

	o.PopulateOptions.Options.BaseOptions = o.EnvOptions.BaseOptions
	o.PopulateOptions.Options.BatchMode = true
	o.PopulateOptions.Dir = o.EnvOptions.Dir
	o.PopulateOptions.DisableSecretFolder = true

	err = o.ConvertOptions.Validate()
	if err != nil {
		return "", errors.Wrapf(err, "failed to validate options")
	}

	sections := strings.Split(text, resourcesSeparator)

	buf := &strings.Builder{}

	for i, section := range sections {
		if i > 0 {
			buf.WriteString("\n")
			buf.WriteString(resourcesSeparator)
		}
		if IsWhitespaceOrComments(section) {
			buf.WriteString(section)
			continue
		}
		result, err := o.Convert(section)
		if err != nil {
			o.LogError(fmt.Sprintf("failed to convert resource: %s", err.Error()))
		}
		buf.WriteString(result)
	}

	if o.SecretCount > 0 && !o.DisablePopulate {
		err = o.PopulateSecrets()
		if err != nil {
			message := fmt.Sprintf("ERROR: failed to populate external secret store: %s\n", err.Error())
			o.LogError(message)
		}
	}
	return buf.String(), nil
}

func (o *Options) LogError(message string) {
	wrote := false
	f, err := os.OpenFile("jx-secret-postrenderer.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	if err == nil {
		_, err = f.WriteString(message)
		if err == nil {
			wrote = true
		}
	}
	if !wrote {
		fmt.Fprintf(os.Stderr, message)
	}
}

func (o *Options) Convert(text string) (string, error) {
	path := ""
	node, err := yaml2.Parse(text)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse YAML")
	}
	if !secretFilter.Matches(node, path) {
		return text, nil
	}

	secretData, err := o.GetSecretData(node, path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get secret data")
	}

	// lets transform....
	opts, err := o.ConvertOptions.ModifyYAML(node, path)
	if err != nil {
		return text, errors.Wrapf(err, "failed to convert Secret")
	}

	// lets save the modified node
	out, err := node.String()
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal converted YAML")
	}

	result, err := o.CreateSecretPair(out)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create SecretPair")
	}
	o.PopulateOptions.Results = append(o.PopulateOptions.Results, result)

	// populate the secret data so we can lazy populate any external secret store
	if secretData != nil {
		key := scm.Join(opts.Namespace, opts.Name)
		if o.PopulateOptions.HelmSecretValues == nil {
			o.PopulateOptions.HelmSecretValues = map[string]map[string]string{}
		}
		o.PopulateOptions.HelmSecretValues[key] = secretData
	}

	o.SecretCount++
	return out, nil
}

func (o *Options) PopulateSecrets() error {
	o.PopulateOptions.DisableLoadResults = true
	err := o.PopulateOptions.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to populate serets")
	}
	return nil
}

func (o *Options) GetSecretData(node *yaml2.RNode, path string) (map[string]string, error) {
	m := map[string]string{}
	for _, dataPath := range []string{"data", "stringData"} {
		data, err := node.Pipe(yaml2.Lookup(dataPath))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get data for path %s", path)
		}

		if data != nil {
			fields, err := data.Fields()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to find data fields for path %s", path)
			}

			for _, field := range fields {
				//if o.SecretMapping.IsSecretKeyUnsecured(secretName, field) {
				value := kyamls.GetStringField(data, "", field)
				if value == "" {
					continue
				}
				m[field] = value
			}
		}
	}
	return m, nil
}

// CreateSecretPair creates the pair of Secret and ExternalSecret used to populate any missing values
func (o *Options) CreateSecretPair(externalSecretYAML string) (*secretfacade.SecretPair, error) {
	pair := &secretfacade.SecretPair{}
	err := yaml.Unmarshal([]byte(externalSecretYAML), &pair.ExternalSecret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal ExternalSecret YAML")
	}
	return pair, nil
}

// IsWhitespaceOrComments returns true if the text is empty, whitespace or comments only
func IsWhitespaceOrComments(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "--") {
			return false
		}
	}
	return true
}
