package storage

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	registryAuthURL = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull"
	registryBaseURL = "https://registry-1.docker.io/v2"
	baseImagesPath  = "/var/lib/minidocker/images"
)

type ManifestLayer struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}

type PlatformSpec struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

type ManifestDescriptor struct {
	Digest    string       `json:"digest"`
	MediaType string       `json:"mediaType"`
	Platform  PlatformSpec `json:"platform"`
}

type ImageIndex struct {
	SchemaVersion int                  `json:"schemaVersion"`
	MediaType     string               `json:"mediaType"`
	Manifests     []ManifestDescriptor `json:"manifests"`
}

type SingleManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	Layers        []ManifestLayer `json:"layers"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

// PullImage descarga y ensambla una imagen desde Docker Hub.
func PullImage(imageName string) (string, error) {
	repo, tag := parseImageName(imageName)
	cleanRepoName := strings.ReplaceAll(repo, "/", "_")
	targetDir := filepath.Join(baseImagesPath, cleanRepoName+"_"+tag, "rootfs")

	// Si ya existe y contiene binarios, se reutiliza
	if stat, err := os.Stat(filepath.Join(targetDir, "bin")); err == nil && stat.IsDir() {
		return targetDir, nil
	}

	// Limpiar si quedó vacía de un intento anterior
	_ = os.RemoveAll(filepath.Dir(targetDir))

	fmt.Printf("[OCI] Solicitando token y manifiesto para [%s:%s]...\n", repo, tag)

	token, err := getAuthToken(repo)
	if err != nil {
		return "", fmt.Errorf("error autenticando con Docker Hub: %w", err)
	}

	layers, err := resolveLayers(repo, tag, token)
	if err != nil {
		return "", fmt.Errorf("error resolviendo capas: %w", err)
	}

	if len(layers) == 0 {
		return "", fmt.Errorf("el manifiesto no contiene capas para la arquitectura %s", runtime.GOARCH)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	client := &http.Client{}
	for i, layer := range layers {
		shortDigest := layer.Digest
		if len(shortDigest) > 19 {
			shortDigest = shortDigest[:19]
		}
		fmt.Printf("  -> Descargando capa %d/%d (%s)...\n", i+1, len(layers), shortDigest)
		layerURL := fmt.Sprintf("%s/%s/blobs/%s", registryBaseURL, repo, layer.Digest)

		req, _ := http.NewRequest("GET", layerURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("falló descarga de blob %s (status: %d)", layer.Digest, resp.StatusCode)
		}

		if err := extractTarGz(resp.Body, targetDir); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("error extrayendo capa: %w", err)
		}
		resp.Body.Close()
	}

	fmt.Println("[OCI] Imagen descargada y ensamblada correctamente.")
	return targetDir, nil
}

func parseImageName(image string) (string, string) {
	tag := "latest"
	parts := strings.Split(image, ":")
	repo := parts[0]
	if len(parts) > 1 {
		tag = parts[1]
	}
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}
	return repo, tag
}

func getAuthToken(repo string) (string, error) {
	url := fmt.Sprintf(registryAuthURL, repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var t TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", err
	}
	return t.Token, nil
}

// resolveLayers maneja tanto Manifiestos Simples como Manifest Lists / OCI Indexes multi-arquitectura
func resolveLayers(repo, tagOrDigest, token string) ([]ManifestLayer, error) {
	url := fmt.Sprintf("%s/%s/manifests/%s", registryBaseURL, repo, tagOrDigest)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	// Aceptamos tanto formatos v2 individuales como índices OCI / listas
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 1. Intentar interpretar como Manifiesto Simple con capas
	var single SingleManifest
	if err := json.Unmarshal(bodyBytes, &single); err == nil && len(single.Layers) > 0 {
		return single.Layers, nil
	}

	// 2. Si no tiene capas directas, intentar interpretar como Índice Multi-arquitectura
	var idx ImageIndex
	if err := json.Unmarshal(bodyBytes, &idx); err == nil && len(idx.Manifests) > 0 {
		targetArch := runtime.GOARCH // ej: "amd64" o "arm64"
		for _, m := range idx.Manifests {
			if m.Platform.OS == "linux" && m.Platform.Architecture == targetArch {
				// Recursión: obtener el manifiesto específico para linux/amd64
				return resolveLayers(repo, m.Digest, token)
			}
		}
	}

	return nil, fmt.Errorf("no se encontraron capas válidas en la respuesta del registro")
}

func extractTarGz(gzipStream io.Reader, target string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			_ = os.Remove(path)
			outFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("error creando archivo %s: %w", path, err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			_ = os.Remove(path)
			if err := os.Symlink(header.Linkname, path); err != nil {
				return fmt.Errorf("error creando symlink %s -> %s: %w", path, header.Linkname, err)
			}

		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			_ = os.Remove(path)
			oldPath := filepath.Join(target, header.Linkname)
			if err := os.Link(oldPath, path); err != nil {
				return fmt.Errorf("error creando hardlink %s -> %s: %w", path, oldPath, err)
			}
		}
	}
	return nil
}
