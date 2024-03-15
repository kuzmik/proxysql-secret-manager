package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"strings"

	gcsm "cloud.google.com/go/secretmanager/apiv1"
	gcsmpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Embed the files in ./templates during compilation, so that we don't need top
// copy a bunch of files around. The format hasn't changed in years, so little
// risk of having something go wrong here.
//
//go:embed templates
var templatesFS embed.FS

func main() {
	projectID := "your_gcp_project"

	secrets, err := populateSecrets(projectID)
	if err != nil {
		log.Fatalf("Failed to populate secrets: %v", err)
	}

	err = WriteCredentials(secrets)
	if err != nil {
		log.Fatalf("Failed to write credential files: %v", err)
	}
}

// Build a secrets map based on the defined secret names from GCSM, and return it.
func populateSecrets(project string) (map[string]string, error) {
	ctx := context.Background()

	secrets := make(map[string]string)

	// Since I'm using my laptop for now, I don't need this file. But perhaps when this is
	// running in a k8s pod we'll need to specify this.
	// TODO: make this a configuration option?
	// option.WithCredentialsFile("path/to/your/service-account-file.json")
	client, err := gcsm.NewClient(ctx)
	if err != nil {
		return secrets, fmt.Errorf("Error creating GCSM client: %w", err)
	}
	defer client.Close()

	// TODO: make this a configuration option that you define in yaml or something, which
	// can also include the file name to which each secret belongs.
	secretIDs := map[string]string{
		// admin passwords
		"admin_password":   "proxysql-password--admin",
		"cluster_password": "proxysql-password--cluster",
		"datadog_password": "proxysql-password--datadog",
		"radmin_password":  "proxysql-password--radmin",
		// client passwords
		"client_datadog_password": fmt.Sprintf("mysql-%s-usc1-password--datadog", project),
		"client_web":              fmt.Sprintf("mysql-%s-usc1-password--web", project),
		"client_proxysql":         fmt.Sprintf("mysql-%s-usc1-password--proxysql", project),
		"client_temporal":         fmt.Sprintf("mysql-%s-usc1-password--temporal", project),
	}

	for name, id := range secretIDs {
		secret, err := accessSecretVersion(ctx, client, project, id)
		if err != nil {
			return secrets, fmt.Errorf("Failed to access secret %s: %w", name, err)
		}

		secrets[name] = secret
	}

	return secrets, nil
}

// Use the GCSM client to fetch the value of the latest vertsion of the specified secret.
func accessSecretVersion(ctx context.Context, client *gcsm.Client, projectID string, secretID string) (string, error) {
	req := &gcsmpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Error when accessing GCSM secrets: %w", err)
	}

	return string(result.Payload.Data), nil
}

// Use the secrets map to interpolate the values into the template files and write the files
// to ./tmp (for now).
// TODO: add a configuration value to define the target directory.
func WriteCredentials(secrets map[string]string) error {
	templates, err := fs.ReadDir(templatesFS, "templates")
	if err != nil {
		return fmt.Errorf("Failed to read directory: %w", err)
	}

	for _, entry := range templates {
		if !entry.IsDir() {
			// Read the file content.
			fileData, err := fs.ReadFile(templatesFS, "templates/"+entry.Name())
			if err != nil {
				return fmt.Errorf("Failed to read template file: %w", err)
			}

			// Parse the template.
			tmpl, err := template.New(entry.Name()).Parse(string(fileData))
			if err != nil {
				return fmt.Errorf("Failed to parse template: %w", err)
			}

			// Execute the template.
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, secrets); err != nil {
				return fmt.Errorf("Failed to execute template: %w", err)
			}

			// Write the output to a file, using the template name with a .txt extension.
			outputFileName := "tmp/" + strings.TrimSuffix(entry.Name(), ".tmpl") + ".cnf"
			if err := os.WriteFile(outputFileName, buf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("Failed to write to file: %w", err)
			}
		}
	}

	return nil
}
