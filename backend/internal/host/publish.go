package host

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type rcloneEntry struct {
	Path string `json:"Path"`
	Size int64  `json:"Size"`
}

type PublishR2Options struct {
	Dev    bool
	DryRun bool
}

func PublishR2(ws *Workspace, assets string, opts PublishR2Options) error {
	assets = ws.ResolvePath(assets)
	prefix := ""
	if opts.Dev {
		prefix = "dev/"
	}
	envFile := expandHome(os.Getenv("SLIPPIOS_R2_ENV"))
	if envFile == "" {
		envFile = filepath.Join(homeDir(), ".config/slippios/r2.env")
	}
	values, err := readEnvFile(envFile)
	if err != nil {
		return fmt.Errorf("missing %s", envFile)
	}
	required := []string{"R2_BUCKET", "R2_ENDPOINT", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_PUBLIC_URL"}
	for _, name := range required {
		if values[name] == "" {
			return fmt.Errorf("%s is not set in %s", name, envFile)
		}
	}
	if _, err := exec.LookPath("rclone"); err != nil {
		return fmt.Errorf("rclone is required")
	}

	bucket := values["R2_BUCKET"]
	baseArgs := []string{
		"--config", "/dev/null",
		"--s3-provider", "Cloudflare",
		"--s3-access-key-id", values["R2_ACCESS_KEY_ID"],
		"--s3-secret-access-key", values["R2_SECRET_ACCESS_KEY"],
		"--s3-endpoint", values["R2_ENDPOINT"],
		"--s3-region", "auto",
	}

	type upload struct {
		source string
		name   string
	}
	var files []upload
	add := func(source, name string) {
		if !fileExists(source) {
			return
		}
		if name == "" {
			name = filepath.Base(source)
		}
		files = append(files, upload{source: source, name: name})
	}
	for _, pattern := range []string{"slippios_*.nix-store.raw", "slippios_*.nix-store.raw.zsync", "slippios_*.efi"} {
		for _, path := range Glob(assets, pattern) {
			add(path, "")
		}
	}
	add(filepath.Join(assets, "SHA256SUMS.update"), "SHA256SUMS")
	add(filepath.Join(assets, "manifest.json"), "")
	if len(files) == 0 {
		return fmt.Errorf("no release assets found in %s", assets)
	}
	if !fileExists(filepath.Join(assets, "manifest.json")) {
		return fmt.Errorf("missing %s", filepath.Join(assets, "manifest.json"))
	}

	runner := NewRunner(ws.Root)
	listArgs := append([]string{"lsjson", ":s3:" + bucket, "--recursive"}, baseArgs...)
	out, err := runner.Output("rclone", listArgs...)
	if err != nil {
		return fmt.Errorf("cannot list R2 bucket %s: %w", bucket, err)
	}
	var remote []rcloneEntry
	if err := json.Unmarshal([]byte(out), &remote); err != nil {
		return err
	}
	remoteSizes := map[string]int64{}
	var currentTotal int64
	for _, entry := range remote {
		if entry.Size < 0 {
			continue
		}
		remoteSizes[entry.Path] = entry.Size
		currentTotal += entry.Size
	}

	var newTotal, replacedTotal int64
	newNames := map[string]bool{}
	for _, file := range files {
		size, _ := FileSize(file.source)
		newTotal += size
		name := prefix + file.name
		newNames[name] = true
		if existing, ok := remoteSizes[name]; ok {
			replacedTotal += existing
		}
	}
	var obsolete []string
	var obsoleteTotal int64
	for name, size := range remoteSizes {
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
		} else if strings.HasPrefix(name, "dev/") {
			continue
		}
		if !newNames[name] {
			obsolete = append(obsolete, name)
			obsoleteTotal += size
		}
	}
	projected := currentTotal - replacedTotal - obsoleteTotal + newTotal
	limit := int64(10 * 1000 * 1000 * 1000)
	fmt.Printf("R2 usage: current %d B, replacing %d B, upload %d B, projected %d B / %d B\n",
		currentTotal, replacedTotal, newTotal, projected, limit)
	if projected > limit && os.Getenv("SLIPPIOS_R2_ALLOW_OVER_QUOTA") != "1" {
		return fmt.Errorf("R2 bucket would exceed the 10 GB free tier; delete the previous release assets first, or set SLIPPIOS_R2_ALLOW_OVER_QUOTA=1")
	}
	if projected > limit {
		fmt.Fprintln(os.Stderr, "warning: projected R2 usage exceeds the 10 GB free tier")
	}
	if opts.DryRun {
		fmt.Println("Dry run: not uploading or deleting anything.")
		return nil
	}

	for _, file := range files {
		fmt.Printf("Uploading %s\n", file.name)
		args := append([]string{"copyto", file.source, ":s3:" + bucket + "/" + prefix + file.name}, baseArgs...)
		args = append(args, "--s3-no-check-bucket")
		if err := runner.Run("rclone", args...); err != nil {
			return err
		}
	}
	if len(obsolete) > 0 {
		fmt.Printf("Removing %d old release assets\n", len(obsolete))
		for _, name := range obsolete {
			args := append([]string{"deletefile", ":s3:" + bucket + "/" + name}, baseArgs...)
			_ = runner.RunQuiet("rclone", args...)
		}
	}

	publicBase := strings.TrimSuffix(values["R2_PUBLIC_URL"], "/")
	resp, err := manifestClient.Get(publicBase + "/" + prefix + "manifest.json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	manifest, err := decodeManifest(data)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Printf("Published SlippiOS %s\n", manifest.Version)
	fmt.Printf("Manifest: %s/%smanifest.json\n", publicBase, prefix)
	return nil
}

func PublishSourceForge(ws *Workspace, assets, user string) error {
	assets = ws.ResolvePath(assets)
	if user == "" {
		user = os.Getenv("SF_USER")
	}
	if user == "" {
		return fmt.Errorf("usage: publish sourceforge <assets-dir> <sourceforge-user>")
	}
	project := os.Getenv("SF_PROJECT")
	if project == "" {
		project = "slippi-os"
	}
	key := expandHome(os.Getenv("SF_KEY"))
	if key == "" {
		key = filepath.Join(homeDir(), ".ssh/sourceforge_slippios")
	}
	if !fileExists(key) {
		return fmt.Errorf("missing SSH key %s", key)
	}
	manifest, err := ReadManifest(filepath.Join(assets, "manifest.json"))
	if err != nil {
		return err
	}
	version := manifest.Version
	remote := "/home/frs/project/" + project + "/v" + version
	target := user + "@frs.sourceforge.net:" + remote

	files := []string{
		filepath.Join(assets, "slippios-usb-"+version+".img.xz"),
		filepath.Join(assets, "slippios-usb-"+version+"-unsigned.img.xz"),
		filepath.Join(assets, "SHA256SUMS"),
		filepath.Join(assets, "LICENSE"),
		filepath.Join(assets, "NOTICE"),
		filepath.Join(assets, "THIRD_PARTY_LICENSES.md"),
		filepath.Join(assets, "SOURCE"),
		filepath.Join(assets, "README.md"),
		filepath.Join(assets, "LICENSES.tar.gz"),
	}
	minimalSigned := filepath.Join(assets, "slippios-usb-"+version+"-minimal.img.xz")
	minimalUnsigned := filepath.Join(assets, "slippios-usb-"+version+"-minimal-unsigned.img.xz")
	if fileExists(minimalSigned) {
		files = append(files, minimalSigned, minimalUnsigned)
	}
	for _, file := range files {
		if !fileExists(file) {
			return fmt.Errorf("missing %s", file)
		}
	}
	signedFile := filepath.Join(assets, "slippios-usb-"+version+".img.xz")
	var support []string
	for _, file := range files {
		if file != signedFile {
			support = append(support, file)
		}
	}

	if err := sftpBatch(key, user, "ls /home/frs/project/"+project+"\n"); err != nil {
		return fmt.Errorf("SourceForge project '%s' not found.\nCreate it at https://sourceforge.net/create/ and wait for approval", project)
	}
	batch := "-mkdir " + remote + "\nls " + remote + "\n"
	if err := sftpBatch(key, user, batch); err != nil {
		return fmt.Errorf("could not create %s.\nCreate the v%s folder in the SourceForge file manager, then rerun", remote, version)
	}

	runner := NewRunner(ws.Root)
	sshArgs := []string{"-i", key, "-o", "StrictHostKeyChecking=accept-new"}
	rsyncSupport := []string{"-avP", "-e", "ssh " + strings.Join(sshArgs, " ")}
	rsyncSupport = append(rsyncSupport, support...)
	rsyncSupport = append(rsyncSupport, target+"/")
	if err := runner.Run("rsync", rsyncSupport...); err != nil {
		return err
	}
	rsyncSigned := []string{"-avP", "--no-times", "-e", "ssh " + strings.Join(sshArgs, " "), signedFile, target + "/"}
	if err := runner.Run("rsync", rsyncSigned...); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Uploaded to https://sourceforge.net/projects/%s/files/v%s/\n", project, version)
	fmt.Println("The signed image was uploaded last; set it as the default download in the SourceForge file manager.")
	return nil
}

func sftpBatch(key, user, batch string) error {
	cmd := exec.Command("sftp", "-q", "-b", "-", "-i", key, "-o", "StrictHostKeyChecking=accept-new", user+"@frs.sourceforge.net")
	cmd.Stdin = strings.NewReader(batch)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		values[key] = value
	}
	return values, nil
}
