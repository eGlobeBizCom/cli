package buildpack

import (
	"fmt"
	"strconv"

	"github.com/cloudfoundry/cli/cf/flags"
	. "github.com/cloudfoundry/cli/cf/i18n"

	"github.com/cloudfoundry/cli/cf"
	"github.com/cloudfoundry/cli/cf/api"
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/cf/models"
	"github.com/cloudfoundry/cli/cf/requirements"
	"github.com/cloudfoundry/cli/cf/terminal"
)

type CreateBuildpack struct {
	ui                terminal.UI
	buildpackRepo     api.BuildpackRepository
	buildpackBitsRepo api.BuildpackBitsRepository
}

func init() {
	commandregistry.Register(&CreateBuildpack{})
}

func (cmd *CreateBuildpack) MetaData() commandregistry.CommandMetadata {
	fs := make(map[string]flags.FlagSet)
	fs["enable"] = &flags.BoolFlag{Name: "enable", Usage: T("Enable the buildpack to be used for staging")}
	fs["disable"] = &flags.BoolFlag{Name: "disable", Usage: T("Disable the buildpack from being used for staging")}

	return commandregistry.CommandMetadata{
		Name:        "create-buildpack",
		Description: T("Create a buildpack"),
		Usage: []string{
			T("CF_NAME create-buildpack BUILDPACK PATH POSITION [--enable|--disable]"),
			T("\n\nTIP:\n"),
			T("   Path should be a zip file, a url to a zip file, or a local directory. Position is a positive integer, sets priority, and is sorted from lowest to highest."),
		},
		Flags:     fs,
		TotalArgs: 3,
	}
}

func (cmd *CreateBuildpack) Requirements(requirementsFactory requirements.Factory, fc flags.FlagContext) []requirements.Requirement {
	if len(fc.Args()) != 3 {
		cmd.ui.Failed(T("Incorrect Usage. Requires buildpack_name, path and position as arguments\n\n") + commandregistry.Commands.CommandUsage("create-buildpack"))
	}

	reqs := []requirements.Requirement{
		requirementsFactory.NewLoginRequirement(),
	}

	return reqs
}

func (cmd *CreateBuildpack) SetDependency(deps commandregistry.Dependency, pluginCall bool) commandregistry.Command {
	cmd.ui = deps.UI
	cmd.buildpackRepo = deps.RepoLocator.GetBuildpackRepository()
	cmd.buildpackBitsRepo = deps.RepoLocator.GetBuildpackBitsRepository()
	return cmd
}

func (cmd *CreateBuildpack) Execute(c flags.FlagContext) error {
	buildpackName := c.Args()[0]

	cmd.ui.Say(T("Creating buildpack {{.BuildpackName}}...", map[string]interface{}{"BuildpackName": terminal.EntityNameColor(buildpackName)}))

	buildpack, err := cmd.createBuildpack(buildpackName, c)

	if err != nil {
		if httpErr, ok := err.(errors.HTTPError); ok && httpErr.ErrorCode() == errors.BuildpackNameTaken {
			cmd.ui.Ok()
			cmd.ui.Warn(T("Buildpack {{.BuildpackName}} already exists", map[string]interface{}{"BuildpackName": buildpackName}))
			cmd.ui.Say(T("TIP: use '{{.CfUpdateBuildpackCommand}}' to update this buildpack", map[string]interface{}{"CfUpdateBuildpackCommand": terminal.CommandColor(cf.Name + " " + "update-buildpack")}))
		} else {
			return err
		}
		return nil
	}
	cmd.ui.Ok()
	cmd.ui.Say("")

	cmd.ui.Say(T("Uploading buildpack {{.BuildpackName}}...", map[string]interface{}{"BuildpackName": terminal.EntityNameColor(buildpackName)}))

	err = cmd.buildpackBitsRepo.UploadBuildpack(buildpack, c.Args()[1])
	if err != nil {
		return err
	}

	cmd.ui.Ok()
	return nil
}

func (cmd CreateBuildpack) createBuildpack(buildpackName string, c flags.FlagContext) (buildpack models.Buildpack, apiErr error) {
	position, err := strconv.Atoi(c.Args()[2])
	if err != nil {
		apiErr = fmt.Errorf(T("Error {{.ErrorDescription}} is being passed in as the argument for 'Position' but 'Position' requires an integer.  For more syntax help, see `cf create-buildpack -h`.", map[string]interface{}{"ErrorDescription": c.Args()[2]}))
		return
	}

	enabled := c.Bool("enable")
	disabled := c.Bool("disable")
	if enabled && disabled {
		apiErr = errors.New(T("Cannot specify both {{.Enabled}} and {{.Disabled}}.", map[string]interface{}{
			"Enabled":  "enabled",
			"Disabled": "disabled",
		}))
		return
	}

	var enableOption *bool
	if enabled {
		enableOption = &enabled
	}
	if disabled {
		disabled = false
		enableOption = &disabled
	}

	buildpack, apiErr = cmd.buildpackRepo.Create(buildpackName, &position, enableOption, nil)

	return
}
