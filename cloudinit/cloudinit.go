package cloudinit

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha4"
)

const (
	standardInitCommand = "etcdadm init"
	standardJoinCommand = "etcdadm join %s"
	// sentinelFileCommand writes a file to /run/cluster-api to signal successful Kubernetes bootstrapping in a way that
	// works both for Linux and Windows OS.
	sentinelFileCommand            = "echo success > /run/cluster-api/bootstrap-success.complete"
	retriableJoinScriptName        = "/usr/local/bin/kubeadm-bootstrap-script"
	retriableJoinScriptOwner       = "root"
	retriableJoinScriptPermissions = "0755"
	cloudConfigHeader              = `## template: jinja
#cloud-config
`
)

var (
	defaultTemplateFuncMap = template.FuncMap{
		"Indent": templateYAMLIndent,
	}
)

func templateYAMLIndent(i int, input string) string {
	split := strings.Split(input, "\n")
	ident := "\n" + strings.Repeat(" ", i)
	return strings.Repeat(" ", i) + strings.Join(split, ident)
}

// BaseUserData is shared across all the various types of files written to disk.
type BaseUserData struct {
	Header               string
	PreEtcdadmCommands   []string
	PostEtcdadmCommands  []string
	AdditionalFiles      []bootstrapv1.File
	WriteFiles           []bootstrapv1.File
	Users                []bootstrapv1.User
	NTP                  *bootstrapv1.NTP
	DiskSetup            *bootstrapv1.DiskSetup
	Mounts               []bootstrapv1.MountPoints
	ControlPlane         bool
	SentinelFileCommand  string
}

func generate(kind string, tpl string, data interface{}) ([]byte, error) {
	tm := template.New(kind).Funcs(defaultTemplateFuncMap)
	if _, err := tm.Parse(filesTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse files template")
	}

	if _, err := tm.Parse(commandsTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse commands template")
	}

	if _, err := tm.Parse(ntpTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse ntp template")
	}

	if _, err := tm.Parse(usersTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse users template")
	}

	if _, err := tm.Parse(diskSetupTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse disk setup template")
	}

	if _, err := tm.Parse(fsSetupTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse fs setup template")
	}

	if _, err := tm.Parse(mountsTemplate); err != nil {
		return nil, errors.Wrap(err, "failed to parse mounts template")
	}

	t, err := tm.Parse(tpl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s template", kind)
	}

	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return nil, errors.Wrapf(err, "failed to generate %s template", kind)
	}

	return out.Bytes(), nil
}

func (input *BaseUserData) prepare() error {
	input.Header = cloudConfigHeader
	input.WriteFiles = append(input.WriteFiles, input.AdditionalFiles...)
	//input.EtcdadmJoinCommand = fmt.Sprintf(standardJoinCommand, input.JoinAddress)
	input.SentinelFileCommand = sentinelFileCommand
	return nil
}
