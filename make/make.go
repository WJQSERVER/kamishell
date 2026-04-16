package make

import (
	"fmt"
	"io"
	"kamishell/builtin"
	"kamishell/core"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type Target struct {
	Name      string
	Sources   []string
	Package   string
	DependsOn []string
	IsLibrary bool
	BuildEnv  map[string]string
}

type Project struct {
	Name    string
	Targets map[string]*Target
}

var currentProject *Project

func parseMakeArgs(args []string) (filename string, params map[string]core.Object) {
	params = make(map[string]core.Object)

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			kv := strings.TrimPrefix(arg, "--")
			if idx := strings.Index(kv, "="); idx > 0 {
				key := kv[:idx]
				value := kv[idx+1:]
				params[key] = inferType(value)
			}
		} else if filename == "" && !strings.HasPrefix(arg, "-") {
			filename = arg
		}
	}

	return filename, params
}

func inferType(value string) core.Object {
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return &core.String{Value: value[1 : len(value)-1]}
	}

	if value == "true" {
		return core.TRUE
	}
	if value == "false" {
		return core.FALSE
	}

	if strings.Contains(value, ".") {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return &core.Float{Value: f}
		}
	}

	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return core.GetInteger(i)
	}

	return &core.String{Value: value}
}

// Make implements the 'make' command.
func Make(args []string, env builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	// 1. Check for help
	if len(args) > 0 && args[0] == "help" {
		printMakeHelp(stdout)
		return 0
	}

	// 2. Parse arguments: filename and --key=value params
	filename, params := parseMakeArgs(args)

	// 3. Initialize project state
	currentProject = &Project{
		Targets: make(map[string]*Target),
	}

	// 4. Look for project files (.km)
	if filename == "" {
		// Default to search for files with .km suffix
		files, err := os.ReadDir(".")
		if err != nil {
			fmt.Fprintf(stderr, "Error reading current directory: %v\n", err)
			return 1
		}

		var kmFiles []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".km") {
				kmFiles = append(kmFiles, f.Name())
			}
		}

		if len(kmFiles) == 0 {
			fmt.Fprintf(stderr, "Error: no .km file found in current directory\n")
			return 1
		}

		if len(kmFiles) == 1 {
			filename = kmFiles[0]
		} else {
			// Prioritize specific names
			for _, name := range []string{"Kami.km", "build.km"} {
				for _, f := range kmFiles {
					if f == name {
						filename = f
						break
					}
				}
				if filename != "" {
					break
				}
			}

			if filename == "" {
				fmt.Fprintf(stderr, "Error: multiple .km files found. Please specify one: %v\n", kmFiles)
				return 1
			}
		}
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "Error: %s not found\n", filename)
		return 1
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading %s: %v\n", filename, err)
		return 1
	}

	// 4. Create a special environment for build script
	coreEnv, ok := env.(*core.Environment)
	if !ok {
		fmt.Fprintf(stderr, "Error: environment is not a core.Environment\n")
		return 1
	}

	buildEnv := core.NewScriptEnvironment(coreEnv)
	for k, v := range params {
		buildEnv.SetObject("param."+k, v)
	}
	restoreBuiltins := registerBuildFunctions()
	defer restoreBuiltins()

	// 5. Run the script
	l := core.NewLexer(string(content))
	p := core.NewParser(l)
	prog := p.ParseProgram()
	core.EvalWithIO(prog, buildEnv, stdin, stdout, stderr)

	// 6. Execute build based on the collected state
	if currentProject.Name != "" {
		fmt.Fprintf(stdout, "Building project: %s\n", currentProject.Name)
	} else {
		fmt.Fprintf(stdout, "Building project...\n")
	}

	for _, target := range currentProject.Targets {
		err := buildTarget(target, stdout, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "Build failed for target %s: %v\n", target.Name, err)
			return 1
		}
	}

	return 0
}

func printMakeHelp(w io.Writer) {
	fmt.Fprintln(w, "\033[1;36mKAMI MAKE - 基于 .km 脚本的构建系统\033[0m")
	fmt.Fprintln(w, "--------------------------------------------------")
	fmt.Fprintln(w, "使用方法:")
	fmt.Fprintln(w, "  make            - 自动寻找并运行 .km 脚本")
	fmt.Fprintln(w, "  make <file.km>  - 运行指定的构建脚本")
	fmt.Fprintln(w, "  make help       - 显示此帮助信息")
	fmt.Fprintln(w, "  make --key=value  - 传递参数给脚本")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "\033[1;33m参数传递:\033[0m")
	fmt.Fprintln(w, "  --name=\"value\"  字符串参数")
	fmt.Fprintln(w, "  --count=123     整数参数")
	fmt.Fprintln(w, "  --flag=true     布尔参数")
	fmt.Fprintln(w, "  --ratio=3.14    浮点参数")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  脚本中使用: param.Get(\"name\")")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "\033[1;33m搜寻规则:\033[0m")
	fmt.Fprintln(w, "  如果不指定文件名，make 会在当前目录下寻找所有 .km 文件。")
	fmt.Fprintln(w, "  - 如果只有一个 .km 文件，直接使用。")
	fmt.Fprintln(w, "  - 如果有多个，优先寻找 'Kami.km' 或 'build.km'。")
	fmt.Fprintln(w, "  - 否则，您需要显式指定文件名。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "\033[1;33m脚本语法 (.km):\033[0m")
	fmt.Fprintln(w, "  使用 Kamishell 原生语法。您可以定义变量、循环和函数。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  \033[1;32mproject \033[0m<name>")
	fmt.Fprintln(w, "    定义项目名称。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  \033[1;32madd_executable \033[0m<target> <sources...>")
	fmt.Fprintln(w, "    定义一个可执行文件目标。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  \033[1;32madd_library \033[0m<target> <sources...>")
	fmt.Fprintln(w, "    定义一个库文件目标。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  \033[1;32mtarget_link_libraries \033[0m<target> <libs...>")
	fmt.Fprintln(w, "    为目标指定链接依赖。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  \033[1;32mtarget_env \033[0m<target> <name=value> [name=value ...]")
	fmt.Fprintln(w, "    为指定目标设置构建环境变量。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "\033[1;33m构建变量:\033[0m")
	fmt.Fprintln(w, "  使用 env.Set()/env.Get()/env.Unset() 管理脚本内构建变量作用域。")
	fmt.Fprintln(w, "  add_executable/add_library 会快照当前 env 包中的变量到目标。")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "\033[1;33m示例 (build.km):\033[0m")
	fmt.Fprintln(w, "  project \"my_app\"")
	fmt.Fprintln(w, "  ")
	fmt.Fprintln(w, "  # 使用变量管理源码")
	fmt.Fprintln(w, "  src := \"main.go api.go utils.go\"")
	fmt.Fprintln(w, "  ")
	fmt.Fprintln(w, "  env.Set(\"GOOS\", \"windows\")")
	fmt.Fprintln(w, "  env.Set(\"GOARCH\", \"amd64\")")
	fmt.Fprintln(w, "  env.Set(\"CGO_ENABLED\", \"0\")")
	fmt.Fprintln(w, "  ")
	fmt.Fprintln(w, "  add_executable \"app_bin\" $src")
	fmt.Fprintln(w, "--------------------------------------------------")
}

func registerBuildFunctions() func() {
	names := []string{"project", "add_executable", "add_library", "target_link_libraries", "target_env"}
	previous := make(map[string]*builtin.BuiltinCommand, len(names))
	for _, name := range names {
		previous[name] = builtin.Builtins[name]
	}

	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: "project",
		Action: func(args []string, e builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if len(args) > 0 {
				currentProject.Name = args[0]
			}
			return 0
		},
	})

	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: "add_executable",
		Action: func(args []string, e builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if len(args) < 2 {
				return 1
			}
			name := args[0]
			sources, pkg, ok := normalizeTargetInputs(args[1:], stderr)
			if !ok {
				return 1
			}
			currentProject.Targets[name] = &Target{
				Name:      name,
				Sources:   sources,
				Package:   pkg,
				IsLibrary: false,
				BuildEnv:  snapshotBuildEnv(e),
			}
			return 0
		},
	})

	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: "add_library",
		Action: func(args []string, e builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if len(args) < 2 {
				return 1
			}
			name := args[0]
			sources, pkg, ok := normalizeTargetInputs(args[1:], stderr)
			if !ok {
				return 1
			}
			currentProject.Targets[name] = &Target{
				Name:      name,
				Sources:   sources,
				Package:   pkg,
				IsLibrary: true,
				BuildEnv:  snapshotBuildEnv(e),
			}
			return 0
		},
	})

	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: "target_link_libraries",
		Action: func(args []string, e builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if len(args) < 2 {
				return 1
			}
			name := args[0]
			libs := args[1:]
			if t, ok := currentProject.Targets[name]; ok {
				t.DependsOn = append(t.DependsOn, libs...)
			}
			return 0
		},
	})

	builtin.RegisterBuiltin(&builtin.BuiltinCommand{
		Name: "target_env",
		Action: func(args []string, e builtin.Environment, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if len(args) < 2 {
				fmt.Fprintf(stderr, "target_env: usage: target_env <target> name=value [name=value ...]\n")
				return 1
			}

			name := args[0]
			target, ok := currentProject.Targets[name]
			if !ok {
				fmt.Fprintf(stderr, "target_env: target %s not found\n", name)
				return 1
			}

			if target.BuildEnv == nil {
				target.BuildEnv = snapshotBuildEnv(e)
			}

			for _, arg := range args[1:] {
				pair := strings.SplitN(arg, "=", 2)
				if len(pair) != 2 || pair[0] == "" {
					fmt.Fprintf(stderr, "target_env: usage: target_env <target> name=value [name=value ...]\n")
					return 1
				}
				setEnvValue(target.BuildEnv, pair[0], pair[1])
			}

			return 0
		},
	})

	return func() {
		for _, name := range names {
			if previous[name] == nil {
				delete(builtin.Builtins, name)
				continue
			}
			builtin.Builtins[name] = previous[name]
		}
	}
}

func buildTarget(t *Target, stdout, stderr io.Writer) error {
	typeStr := "Executable"
	if t.IsLibrary {
		typeStr = "Library"
	}
	fmt.Fprintf(stdout, "  Target: %s (%s)\n", t.Name, typeStr)

	cmd := newBuildCommand(t)
	fmt.Fprintf(stdout, "  Running: %s\n", strings.Join(cmd.Args, " "))
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func newBuildCommand(t *Target) *exec.Cmd {
	args := []string{"build", "-o", targetOutputName(t)}
	if t.Package != "" {
		args = append(args, t.Package)
	} else {
		args = append(args, t.Sources...)
	}

	cmd := exec.Command("go", args...)
	cmd.Env = envListFromMap(effectiveBuildEnv(t))
	return cmd
}

func normalizeTargetInputs(inputs []string, stderr io.Writer) ([]string, string, bool) {
	if len(inputs) == 0 {
		fmt.Fprintln(stderr, "target sources cannot be empty")
		return nil, "", false
	}

	if len(inputs) == 1 && isPackageSource(inputs[0]) {
		return nil, inputs[0], true
	}

	if slices.ContainsFunc(inputs, isPackageSource) {
		fmt.Fprintln(stderr, "package source '.' cannot be mixed with explicit source files")
		return nil, "", false
	}

	return inputs, "", true
}

func isPackageSource(input string) bool {
	return input == "."
}

func effectiveBuildEnv(t *Target) map[string]string {
	if t.BuildEnv == nil {
		return snapshotProcessEnv()
	}
	return t.BuildEnv
}

func targetOutputName(t *Target) string {
	outputName := t.Name
	if targetGOOS(t) == "windows" && !t.IsLibrary && !strings.HasSuffix(strings.ToLower(outputName), ".exe") {
		outputName += ".exe"
	}
	return outputName
}

func snapshotBuildEnv(env builtin.Environment) map[string]string {
	snapshot := snapshotProcessEnv()

	coreEnv, ok := env.(*core.Environment)
	if !ok {
		return snapshot
	}

	for key, value := range coreEnv.PackageSnapshot("env") {
		setEnvValue(snapshot, key, value)
	}

	return snapshot
}

func snapshotProcessEnv() map[string]string {
	snapshot := make(map[string]string)
	for _, entry := range os.Environ() {
		pair := strings.SplitN(entry, "=", 2)
		if len(pair) == 0 || pair[0] == "" {
			continue
		}

		value := ""
		if len(pair) == 2 {
			value = pair[1]
		}

		setEnvValue(snapshot, pair[0], value)
	}
	return snapshot
}

func envListFromMap(envMap map[string]string) []string {
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	envList := make([]string, 0, len(keys))
	for _, key := range keys {
		envList = append(envList, key+"="+envMap[key])
	}
	return envList
}

func setEnvValue(envMap map[string]string, key, value string) {
	if key == "" || strings.Contains(key, "=") {
		return
	}

	if runtime.GOOS == "windows" {
		for existingKey := range envMap {
			if strings.EqualFold(existingKey, key) && existingKey != key {
				delete(envMap, existingKey)
				break
			}
		}
	}

	envMap[key] = value
}

func getEnvValue(envMap map[string]string, key string) (string, bool) {
	if value, ok := envMap[key]; ok {
		return value, true
	}

	if runtime.GOOS == "windows" {
		for existingKey, value := range envMap {
			if strings.EqualFold(existingKey, key) {
				return value, true
			}
		}
	}

	return "", false
}

func targetGOOS(t *Target) string {
	if value, ok := getEnvValue(effectiveBuildEnv(t), "GOOS"); ok && value != "" {
		return strings.ToLower(value)
	}
	return runtime.GOOS
}
