package exec

import (
	"errors"
	"fmt"
	c "github.com/cloudposse/atmos/pkg/config"
	g "github.com/cloudposse/atmos/pkg/globals"
	s "github.com/cloudposse/atmos/pkg/stack"
	"github.com/cloudposse/atmos/pkg/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var (
	commonFlags = []string{
		"--stack",
		"-s",
		"--dry-run",
		"--kubeconfig-path",
		g.TerraformDirFlag,
		g.HelmfileDirFlag,
		g.ConfigDirFlag,
		g.StackDirFlag,
		g.BasePathFlag,
		g.GlobalOptionsFlag,
		g.DeployRunInitFlag,
		g.AutoGenerateBackendFileFlag,
		g.FromPlanFlag,
		g.HelpFlag1,
		g.HelpFlag2,
	}
)

// findComponentConfig finds component config sections
func findComponentConfig(
	stack string,
	stacksMap map[string]interface{},
	componentType string,
	component string,
) (map[string]interface{},
	map[interface{}]interface{},
	map[interface{}]interface{},
	map[interface{}]interface{},
	string,
	string,
	string,
	[]string,
	bool,
	map[interface{}]interface{},
	error,
) {

	var stackSection map[interface{}]interface{}
	var componentsSection map[string]interface{}
	var componentTypeSection map[string]interface{}
	var componentSection map[string]interface{}
	var componentVarsSection map[interface{}]interface{}
	var componentEnvSection map[interface{}]interface{}
	var componentBackendSection map[interface{}]interface{}
	var componentBackendType string
	var baseComponentName string
	var command string
	var componentInheritanceChain []string
	var componentIsAbstract bool
	var componentMetadata map[interface{}]interface{}
	var ok bool

	if len(stack) == 0 {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New("stack must be provided and must not be empty")
	}
	if len(component) == 0 {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New("component must be provided and must not be empty")
	}
	if len(componentType) == 0 {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New("component type must be provided and must not be empty")
	}
	if stackSection, ok = stacksMap[stack].(map[interface{}]interface{}); !ok {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New(fmt.Sprintf("Stack '%s' does not exist", stack))
	}
	if componentsSection, ok = stackSection["components"].(map[string]interface{}); !ok {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New(fmt.Sprintf("'components' section is missing in the stack '%s'", stack))
	}
	if componentTypeSection, ok = componentsSection[componentType].(map[string]interface{}); !ok {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New(fmt.Sprintf("'components/%s' section is missing in the stack '%s'", componentType, stack))
	}
	if componentSection, ok = componentTypeSection[component].(map[string]interface{}); !ok {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New(fmt.Sprintf("Invalid or missing configuration for the component '%s' in the stack '%s'", component, stack))
	}
	if componentVarsSection, ok = componentSection["vars"].(map[interface{}]interface{}); !ok {
		return nil, nil, nil, nil, "", "", "", nil, false, nil, errors.New(fmt.Sprintf("Missing 'vars' section for the component '%s' in the stack '%s'", component, stack))
	}
	if componentBackendSection, ok = componentSection["backend"].(map[interface{}]interface{}); !ok {
		componentBackendSection = nil
	}
	if componentBackendType, ok = componentSection["backend_type"].(string); !ok {
		componentBackendType = ""
	}
	if command, ok = componentSection["command"].(string); !ok {
		command = ""
	}
	if componentEnvSection, ok = componentSection["env"].(map[interface{}]interface{}); !ok {
		componentEnvSection = map[interface{}]interface{}{}
	}
	if componentInheritanceChain, ok = componentSection["inheritance"].([]string); !ok {
		componentInheritanceChain = []string{}
	}
	if baseComponentName, ok = componentSection["component"].(string); !ok {
		baseComponentName = ""
	}
	if componentMetadataSection, componentMetadataSectionExists := componentSection["metadata"]; componentMetadataSectionExists {
		componentMetadata = componentMetadataSection.(map[interface{}]interface{})
		if componentMetadataType, componentMetadataTypeAttributeExists := componentMetadata["type"].(string); componentMetadataTypeAttributeExists {
			if componentMetadataType == "abstract" {
				componentIsAbstract = true
			}
		}
		if componentMetadataComponent, componentMetadataComponentExists := componentMetadata["component"].(string); componentMetadataComponentExists {
			baseComponentName = componentMetadataComponent
		}
	}

	return componentSection,
		componentVarsSection,
		componentEnvSection,
		componentBackendSection,
		componentBackendType,
		baseComponentName,
		command,
		componentInheritanceChain,
		componentIsAbstract,
		componentMetadata,
		nil
}

// processConfigAndStacks processes CLI config and stacks
func processConfigAndStacks(componentType string, cmd *cobra.Command, args []string) (c.ConfigAndStacksInfo, error) {
	var configAndStacksInfo c.ConfigAndStacksInfo

	if len(args) < 1 {
		return configAndStacksInfo, errors.New("invalid number of arguments")
	}

	cmd.DisableFlagParsing = false

	err := cmd.ParseFlags(args)
	if err != nil {
		return configAndStacksInfo, err
	}

	argsAndFlagsInfo, err := processArgsAndFlags(args)
	if err != nil {
		return configAndStacksInfo, err
	}

	configAndStacksInfo.AdditionalArgsAndFlags = argsAndFlagsInfo.AdditionalArgsAndFlags
	configAndStacksInfo.SubCommand = argsAndFlagsInfo.SubCommand
	configAndStacksInfo.ComponentFromArg = argsAndFlagsInfo.ComponentFromArg
	configAndStacksInfo.GlobalOptions = argsAndFlagsInfo.GlobalOptions
	configAndStacksInfo.BasePath = argsAndFlagsInfo.BasePath
	configAndStacksInfo.TerraformDir = argsAndFlagsInfo.TerraformDir
	configAndStacksInfo.HelmfileDir = argsAndFlagsInfo.HelmfileDir
	configAndStacksInfo.StacksDir = argsAndFlagsInfo.StacksDir
	configAndStacksInfo.ConfigDir = argsAndFlagsInfo.ConfigDir
	configAndStacksInfo.DeployRunInit = argsAndFlagsInfo.DeployRunInit
	configAndStacksInfo.AutoGenerateBackendFile = argsAndFlagsInfo.AutoGenerateBackendFile
	configAndStacksInfo.UseTerraformPlan = argsAndFlagsInfo.UseTerraformPlan
	configAndStacksInfo.NeedHelp = argsAndFlagsInfo.NeedHelp

	// Check if `-h` or `--help` flags are specified
	if argsAndFlagsInfo.NeedHelp == true {
		err = processHelp(componentType, argsAndFlagsInfo.SubCommand)
		if err != nil {
			return configAndStacksInfo, err
		}
		return configAndStacksInfo, nil
	}

	flags := cmd.Flags()
	configAndStacksInfo.Stack, err = flags.GetString("stack")
	if err != nil {
		return configAndStacksInfo, err
	}

	// Check if stack was provided
	if len(configAndStacksInfo.Stack) < 1 {
		message := fmt.Sprintf("'stack' is required. Usage: atmos %s <command> <component> -s <stack>", componentType)
		return configAndStacksInfo, errors.New(message)
	}

	// Check if component was provided
	if len(configAndStacksInfo.ComponentFromArg) < 1 {
		message := fmt.Sprintf("'component' is required. Usage: atmos %s <command> <component> <arguments_and_flags>", componentType)
		return configAndStacksInfo, errors.New(message)
	}

	// Process and merge CLI configurations
	err = c.InitConfig()
	if err != nil {
		return configAndStacksInfo, err
	}

	err = c.ProcessConfig(configAndStacksInfo)
	if err != nil {
		return configAndStacksInfo, err
	}

	// Process stack config file(s)
	_, stacksMap, err := s.ProcessYAMLConfigFiles(
		c.ProcessedConfig.StacksBaseAbsolutePath,
		c.ProcessedConfig.StackConfigFilesAbsolutePaths,
		false,
		true)

	if err != nil {
		return configAndStacksInfo, err
	}

	// Print the stack config files
	if g.LogVerbose {
		fmt.Println()
		var msg string
		if c.ProcessedConfig.StackType == "Directory" {
			msg = "Found the config file for the provided stack:"
		} else {
			msg = "Found config files:"
		}
		color.Cyan(msg)
		err = utils.PrintAsYAML(c.ProcessedConfig.StackConfigFilesRelativePaths)
		if err != nil {
			return configAndStacksInfo, err
		}
	}

	if len(c.Config.Stacks.NamePattern) < 1 {
		return configAndStacksInfo,
			errors.New("stack name pattern must be provided in 'stacks.name_pattern' config or 'ATMOS_STACKS_NAME_PATTERN' ENV variable")
	}

	stackNamePatternParts := strings.Split(c.Config.Stacks.NamePattern, "-")

	// Check and process stacks
	if c.ProcessedConfig.StackType == "Directory" {
		_, configAndStacksInfo.ComponentVarsSection,
			configAndStacksInfo.ComponentEnvSection,
			configAndStacksInfo.ComponentBackendSection,
			configAndStacksInfo.ComponentBackendType,
			configAndStacksInfo.BaseComponentPath,
			configAndStacksInfo.Command,
			configAndStacksInfo.ComponentInheritanceChain,
			configAndStacksInfo.ComponentIsAbstract,
			configAndStacksInfo.ComponentMetadataSection,
			err = findComponentConfig(configAndStacksInfo.Stack, stacksMap, componentType, configAndStacksInfo.ComponentFromArg)
		if err != nil {
			return configAndStacksInfo, err
		}

		configAndStacksInfo.ComponentEnvList = convertEnvVars(configAndStacksInfo.ComponentEnvSection)
	} else {
		if g.LogVerbose {
			color.Cyan("Searching for stack config where the component '%s' is defined\n", configAndStacksInfo.ComponentFromArg)
		}

		stackParts := strings.Split(configAndStacksInfo.Stack, "-")
		if len(stackParts) != len(stackNamePatternParts) {
			return configAndStacksInfo,
				errors.New(fmt.Sprintf("Stack '%s' does not match the stack name pattern '%s'",
					configAndStacksInfo.Stack,
					c.Config.Stacks.NamePattern))
		}

		var tenant string
		var environment string
		var stage string
		var tenantFound bool
		var environmentFound bool
		var stageFound bool

		for i, part := range stackNamePatternParts {
			if part == "{tenant}" {
				tenant = stackParts[i]
			} else if part == "{environment}" {
				environment = stackParts[i]
			} else if part == "{stage}" {
				stage = stackParts[i]
			}
		}

		for stackName := range stacksMap {
			_, configAndStacksInfo.ComponentVarsSection,
				configAndStacksInfo.ComponentEnvSection,
				configAndStacksInfo.ComponentBackendSection,
				configAndStacksInfo.ComponentBackendType,
				configAndStacksInfo.BaseComponentPath,
				configAndStacksInfo.Command,
				configAndStacksInfo.ComponentInheritanceChain,
				configAndStacksInfo.ComponentIsAbstract,
				configAndStacksInfo.ComponentMetadataSection,
				err = findComponentConfig(stackName, stacksMap, componentType, configAndStacksInfo.ComponentFromArg)
			if err != nil {
				continue
			}

			configAndStacksInfo.ComponentEnvList = convertEnvVars(configAndStacksInfo.ComponentEnvSection)

			tenantFound = true
			environmentFound = true
			stageFound = true

			// Search for tenant in stack
			if len(tenant) > 0 {
				if tenantInStack, ok := configAndStacksInfo.ComponentVarsSection["tenant"].(string); !ok || tenantInStack != tenant {
					tenantFound = false
				}
			}

			// Search for environment in stack
			if len(environment) > 0 {
				if environmentInStack, ok := configAndStacksInfo.ComponentVarsSection["environment"].(string); !ok || environmentInStack != environment {
					environmentFound = false
				}
			}

			// Search for stage in stack
			if len(stage) > 0 {
				if stageInStack, ok := configAndStacksInfo.ComponentVarsSection["stage"].(string); !ok || stageInStack != stage {
					stageFound = false
				}
			}

			if tenantFound == true && environmentFound == true && stageFound == true {
				if g.LogVerbose {
					color.Green("Found stack config for the component '%s' in the stack '%s'\n\n", configAndStacksInfo.ComponentFromArg, stackName)
				}
				configAndStacksInfo.Stack = stackName
				break
			}
		}

		if tenantFound == false || environmentFound == false || stageFound == false {
			return configAndStacksInfo,
				errors.New(fmt.Sprintf("\nCould not find config for the component '%s' in the stack '%s'.\n"+
					"Check that all attributes in the stack name pattern '%s' are defined in the stack config files.\n"+
					"Are the component and stack names correct? Did you forget an import?",
					configAndStacksInfo.ComponentFromArg,
					configAndStacksInfo.Stack,
					c.Config.Stacks.NamePattern,
				))
		}
	}

	if len(configAndStacksInfo.Command) == 0 {
		configAndStacksInfo.Command = componentType
	}

	color.Cyan("\nVariables for the component '%s' in the stack '%s':\n\n", configAndStacksInfo.ComponentFromArg, configAndStacksInfo.Stack)
	err = utils.PrintAsYAML(configAndStacksInfo.ComponentVarsSection)
	if err != nil {
		return configAndStacksInfo, err
	}

	configAndStacksInfo.ComponentFolderPrefix = ""
	configAndStacksInfo.ComponentNamePrefix = ""

	finalComponentPathParts := strings.Split(configAndStacksInfo.ComponentFromArg, "/")
	finalComponentPathPartsLength := len(finalComponentPathParts)

	if finalComponentPathPartsLength > 1 {
		componentFromArgPartsWithoutLast := finalComponentPathParts[:finalComponentPathPartsLength-1]
		configAndStacksInfo.ComponentFolderPrefix = strings.Join(componentFromArgPartsWithoutLast, "/")
		configAndStacksInfo.ComponentNamePrefix = strings.Join(componentFromArgPartsWithoutLast, "-")
		configAndStacksInfo.Component = finalComponentPathParts[finalComponentPathPartsLength-1]
	} else {
		configAndStacksInfo.Component = configAndStacksInfo.ComponentFromArg
	}

	if len(configAndStacksInfo.BaseComponentPath) > 0 {
		baseComponentPathParts := strings.Split(configAndStacksInfo.BaseComponentPath, "/")
		baseComponentPathPartsLength := len(baseComponentPathParts)
		if baseComponentPathPartsLength > 1 {
			configAndStacksInfo.BaseComponent = baseComponentPathParts[baseComponentPathPartsLength-1]
		} else {
			configAndStacksInfo.BaseComponent = configAndStacksInfo.BaseComponentPath
		}
	}

	// Process context
	configAndStacksInfo.Context = c.GetContextFromVars(configAndStacksInfo.ComponentVarsSection)
	configAndStacksInfo.ContextPrefix, err = c.GetContextPrefix(configAndStacksInfo.Stack, configAndStacksInfo.Context, c.Config.Stacks.NamePattern)
	if err != nil {
		return configAndStacksInfo, err
	}

	// Terraform workspace can be overridden per component in YAML config `metadata.terraform_workspace`
	if componentTerraformWorkspace, componentTerraformWorkspaceExist := configAndStacksInfo.ComponentMetadataSection["terraform_workspace"].(string); componentTerraformWorkspaceExist {
		configAndStacksInfo.TerraformWorkspace = componentTerraformWorkspace
	}

	return configAndStacksInfo, nil
}

// processArgsAndFlags removes common args and flags from the provided list of arguments/flags
func processArgsAndFlags(inputArgsAndFlags []string) (c.ArgsAndFlagsInfo, error) {
	var info c.ArgsAndFlagsInfo
	var additionalArgsAndFlags []string
	var globalOptions []string

	var indexesToRemove []int

	// https://github.com/roboll/helmfile#cli-reference
	var globalOptionsFlagIndex int

	for i, arg := range inputArgsAndFlags {
		if arg == g.GlobalOptionsFlag {
			globalOptionsFlagIndex = i + 1
		} else if strings.HasPrefix(arg+"=", g.GlobalOptionsFlag) {
			globalOptionsFlagIndex = i
		}

		if arg == g.TerraformDirFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.TerraformDir = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.TerraformDirFlag) {
			var terraformDirFlagParts = strings.Split(arg, "=")
			if len(terraformDirFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.TerraformDir = terraformDirFlagParts[1]
		}

		if arg == g.HelmfileDirFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.HelmfileDir = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.HelmfileDirFlag) {
			var helmfileDirFlagParts = strings.Split(arg, "=")
			if len(helmfileDirFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.HelmfileDir = helmfileDirFlagParts[1]
		}

		if arg == g.ConfigDirFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.StacksDir = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.ConfigDirFlag) {
			var configDirFlagParts = strings.Split(arg, "=")
			if len(configDirFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.StacksDir = configDirFlagParts[1]
		}

		if arg == g.StackDirFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.ConfigDir = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.StackDirFlag) {
			var stacksDirFlagParts = strings.Split(arg, "=")
			if len(stacksDirFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.ConfigDir = stacksDirFlagParts[1]
		}

		if arg == g.BasePathFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.BasePath = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.BasePathFlag) {
			var stacksDirFlagParts = strings.Split(arg, "=")
			if len(stacksDirFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.BasePath = stacksDirFlagParts[1]
		}

		if arg == g.DeployRunInitFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.DeployRunInit = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.DeployRunInitFlag) {
			var deployRunInitFlagParts = strings.Split(arg, "=")
			if len(deployRunInitFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.DeployRunInit = deployRunInitFlagParts[1]
		}

		if arg == g.AutoGenerateBackendFileFlag {
			if len(inputArgsAndFlags) <= (i + 1) {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.AutoGenerateBackendFile = inputArgsAndFlags[i+1]
		} else if strings.HasPrefix(arg+"=", g.AutoGenerateBackendFileFlag) {
			var autoGenerateBackendFileFlagParts = strings.Split(arg, "=")
			if len(autoGenerateBackendFileFlagParts) != 2 {
				return info, errors.New(fmt.Sprintf("invalid flag: %s", arg))
			}
			info.AutoGenerateBackendFile = autoGenerateBackendFileFlagParts[1]
		}

		if arg == g.FromPlanFlag {
			info.UseTerraformPlan = true
		}

		if arg == g.HelpFlag1 || arg == g.HelpFlag2 {
			info.NeedHelp = true
		}

		for _, f := range commonFlags {
			if arg == f {
				indexesToRemove = append(indexesToRemove, i)
				indexesToRemove = append(indexesToRemove, i+1)
			} else if strings.HasPrefix(arg, f+"=") {
				indexesToRemove = append(indexesToRemove, i)
			}
		}
	}

	for i, arg := range inputArgsAndFlags {
		if !utils.SliceContainsInt(indexesToRemove, i) {
			additionalArgsAndFlags = append(additionalArgsAndFlags, arg)
		}

		if globalOptionsFlagIndex > 0 && i == globalOptionsFlagIndex {
			if strings.HasPrefix(arg, g.GlobalOptionsFlag+"=") {
				parts := strings.SplitN(arg, "=", 2)
				globalOptions = strings.Split(parts[1], " ")
			} else {
				globalOptions = strings.Split(arg, " ")
			}
		}
	}

	info.GlobalOptions = globalOptions

	if info.NeedHelp == true {
		if len(additionalArgsAndFlags) > 0 {
			info.SubCommand = additionalArgsAndFlags[0]
		}
		return info, nil
	}

	if len(additionalArgsAndFlags) > 1 {
		// Handle the legacy command `terraform write varfile`
		if additionalArgsAndFlags[0] == "write" && additionalArgsAndFlags[1] == "varfile" {
			info.SubCommand = "write varfile"
			info.ComponentFromArg = additionalArgsAndFlags[2]
			info.AdditionalArgsAndFlags = additionalArgsAndFlags[3:]
		} else {
			info.SubCommand = additionalArgsAndFlags[0]
			info.ComponentFromArg = additionalArgsAndFlags[1]
			info.AdditionalArgsAndFlags = additionalArgsAndFlags[2:]
		}
	} else {
		message := "invalid number of arguments. Usage: atmos <command> <component> <arguments_and_flags>"
		return info, errors.New(message)
	}

	return info, nil
}

// execCommand prints and executes the provided command with args and flags
func execCommand(command string, args []string, dir string, env []string) error {
	cmd := exec.Command(command, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	fmt.Println()
	color.Cyan("Executing command:\n")
	fmt.Println(cmd.String())
	return cmd.Run()
}

func generateComponentBackendConfig(backendType string, backendConfig map[interface{}]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"terraform": map[string]interface{}{
			"backend": map[string]interface{}{
				backendType: backendConfig,
			},
		},
	}
}

// convertEnvVars convert ENV vars from a map to a list of strings in the format ["key1=val1", "key2=val2", "key3=val3" ...]
func convertEnvVars(envVarsMap map[interface{}]interface{}) []string {
	res := []string{}
	if envVarsMap != nil {
		for k, v := range envVarsMap {
			res = append(res, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return res
}

// execTerraformShellCommand executes `terraform shell` command by starting a new interactive shell
func execTerraformShellCommand(
	component string,
	stack string,
	componentEnvList []string,
	varFile string,
	workingDir string,
	workspaceName string,
	componentPath string) error {

	componentEnvList = append(componentEnvList, fmt.Sprintf("TF_CLI_ARGS_plan=-var-file=%s", varFile))
	componentEnvList = append(componentEnvList, fmt.Sprintf("TF_CLI_ARGS_apply=-var-file=%s", varFile))
	componentEnvList = append(componentEnvList, fmt.Sprintf("TF_CLI_ARGS_refresh=-var-file=%s", varFile))
	componentEnvList = append(componentEnvList, fmt.Sprintf("TF_CLI_ARGS_import=-var-file=%s", varFile))
	componentEnvList = append(componentEnvList, fmt.Sprintf("TF_CLI_ARGS_destroy=-var-file=%s", varFile))

	fmt.Println()
	color.Cyan("Starting a new interactive shell where you can execute all native Terraform commands (type 'exit' to go back)")
	fmt.Println(fmt.Sprintf("Component: %s", component))
	fmt.Println(fmt.Sprintf("Stack: %s", stack))
	fmt.Println(fmt.Sprintf("Working directory: %s", workingDir))
	fmt.Println(fmt.Sprintf("Terraform workspace: %s", workspaceName))
	fmt.Println()
	color.Cyan("Setting the ENV vars in the shell:\n")
	for _, v := range componentEnvList {
		fmt.Println(v)
	}
	fmt.Println()

	// Transfer stdin, stdout, and stderr to the new process and also set the target directory for the shell to start in
	pa := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   componentPath,
		Env:   append(os.Environ(), componentEnvList...),
	}

	// Start a new shell
	executableName := ""
	if runtime.GOOS == "windows" {
		executableName = "cmd.exe"
	} else {
		executableName = os.Getenv("SHELL")
	}

	if len(executableName) == 0 {
		return errors.New("can't find a shell to execute")
	}

	proc, err := os.StartProcess(executableName, []string{"-fpl"}, &pa)
	if err != nil {
		return err
	}

	// Wait until user exits the shell
	state, err := proc.Wait()
	if err != nil {
		return err
	}

	fmt.Printf("Exited shell: %s\n", state.String())
	return nil
}