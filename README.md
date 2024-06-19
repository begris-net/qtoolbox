[//]: # ()
[//]: # (Copyright &#40;c&#41; 2023 - 2024 Bjoern Beier.)
[//]: # ()
[//]: # (Permission is hereby granted, free of charge, to any person obtaining a copy)
[//]: # (of this software and associated documentation files &#40;the "Software"&#41;, to deal)
[//]: # (in the Software without restriction, including without limitation the rights)
[//]: # (to use, copy, modify, merge, publish, distribute, sublicense, and/or sell)
[//]: # (copies of the Software, and to permit persons to whom the Software is)
[//]: # (furnished to do so, subject to the following conditions:)
[//]: # ()
[//]: # (The above copyright notice and this permission notice shall be included in all)
[//]: # (copies or substantial portions of the Software.)
[//]: # ()
[//]: # (THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR)
[//]: # (IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,)
[//]: # (FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE)
[//]: # (AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER)
[//]: # (LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,)
[//]: # (OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE)
[//]: # (SOFTWARE.)


# qtoolbox
Simple SDKman! inspired SDK and tool manager. 

# Local build command
```shell
for os in linux windows; do echo "Build $os version..."; GOOS=$os go build -ldflags="-X 'main.Version=v0.0.1-beta'" ; done; ./qtoolbox setup --force
```

# Config

## Global config
```yaml
repository-metadata: var/repository.yaml

provider-settings:
  MavenRelease:
    base-url: "https://repo1.maven.org/maven2"
    version-cache-ttl: 24h #default: 24h
  GitHubRelease:
    page-size: 100 #default: 100
    version-cache-ttl: 24h #default: 24h
  GitHubTagsDownloadUrl:
    page-size: 100 #default: 100
    version-cache-ttl: 24h #default: 24h
```

## Repository Metadata
```yaml
update_url: https://toolbox.github.io
#platform: x64
#os: windows

# default os strings:
#aix/ppc64
#android/386
#android/amd64
#android/arm
#android/arm64
#darwin/amd64
#darwin/arm64
#dragonfly/amd64
#freebsd/386
#freebsd/amd64
#freebsd/arm
#freebsd/arm64
#freebsd/riscv64
#illumos/amd64
#ios/amd64
#ios/arm64
#js/wasm
#linux/386
#linux/amd64
#linux/arm
#linux/arm64
#linux/loong64
#linux/mips
#linux/mips64
#linux/mips64le
#linux/mipsle
#linux/ppc64
#linux/ppc64le
#linux/riscv64
#linux/s390x
#netbsd/386
#netbsd/amd64
#netbsd/arm
#netbsd/arm64
#openbsd/386
#openbsd/amd64
#openbsd/arm
#openbsd/arm64
#plan9/386
#plan9/amd64
#plan9/arm
#solaris/amd64
#wasip1/wasm
#windows/386
#windows/amd64
#windows/arm
#windows/arm64

candidates:
  - name: java
    display-name: Java
    description: |
      Java (21.0.1-tem)        https://projects.eclipse.org/projects/adoptium.temurin/

      Java Platform, Standard Edition (or Java SE) is a widely used platform for
      development and deployment of portable code for desktop and server environments.
      Java SE uses the object-oriented Java programming language. It is part of the
      Java software-platform family. Java SE defines a wide range of general-purpose
      APIs – such as Java APIs for the Java Class Library – and also includes the Java
      Language Specification and the Java Virtual Machine Specification.
    export-path: bin # if a subpath needs to be exported
    default-provider-id: zulu # only required, if multiple provider ids are present. Mainly to determine the latest version during the install command without a specific version
    provider:
      zulu:
        id: zulu
        vendor: Azul
        type: Zulu
        endpoint: https://api.azul.com/metadata/v1/zulu/packages/?os={{.OS}}&archive_type={{.ArchiveType}}&java_package_type=jdk&crs_supported=false&support_term=lts&release_type=PSU&latest=true&release_status=ga&availability_types=ca&certifications=tck&page=1&page_size=1000
        settings:
          os-mapping:
            darwin: macosx
            linux: linux-glibc
#           handle alpine linux ==> linux-musl
          arch-mapping:
            amd64: x64
            386: i686
            arm64: aarch64
            arm: aarch32hf
          archivetype-mapping:
            darwin: tar.gz
            linux: tar.gz
            windows: zip
      amazon-corretto-21:
        id: amzn
        vendor: Amazon
        type: GitHubTagsDownloadUrl
        endpoint: corretto/corretto-21
        settings:
          url: https://corretto.aws/downloads/resources/{{.Version}}/amazon-corretto-{{.Version}}-{{.OS}}-{{.Arch}}{{.OSArchiveExt}}
          os-mapping:
            darwin: macosx
          arch-mapping:
            arm64: aarch64
            amd64: x64
          ext-mapping:
            windows: -jdk.zip # if an extension other than the default is required
            linux: .tar.gz # if an extension other than the default is required
            darwin: .tar.gz # if an extension other than the default is required
        # https://corretto.aws/downloads/resources/21.0.1.12.1/amazon-corretto-21.0.1.12.1-linux-aarch64.tar.gz  << linux - arm64
        # https://corretto.aws/downloads/resources/21.0.1.12.1/amazon-corretto-21.0.1.12.1-linux-x64.tar.gz  << linux - amd64
        # https://corretto.aws/downloads/resources/21.0.1.12.1/amazon-corretto-21.0.1.12.1-windows-x64-jdk.zip  <<  - arm64
        # https://corretto.aws/downloads/resources/21.0.1.12.1/amazon-corretto-21.0.1.12.1-macosx-x64.tar.gz  << darwin - arm64
        # https://corretto.aws/downloads/resources/21.0.1.12.1/amazon-corretto-21.0.1.12.1-macosx-aarch64.tar.gz  << darwin - arm64
      amazon-corretto-17:
        id: amzn
        vendor: Amazon
        type: GitHubTagsDownloadUrl
        endpoint: corretto/corretto-17
        settings:
          url: https://corretto.aws/downloads/resources/{{.Version}}/amazon-corretto-{{.Version}}-{{.OS}}-{{.Arch}}{{.OSArchiveExt}}
          os-mapping:
            darwin: macosx
          arch-mapping:
            arm64: aarch64
            amd64: x64
          ext-mapping:
            windows: -jdk.zip # if an extension other than the default is required
            linux: .tar.gz # if an extension other than the default is required
            darwin: .tar.gz # if an extension other than the default is required
      amazon-corretto-11:
        id: amzn
        vendor: Amazon
        type: GitHubTagsDownloadUrl
        endpoint: corretto/corretto-11
        settings:
          url: https://corretto.aws/downloads/resources/{{.Version}}/amazon-corretto-{{.Version}}-{{.OS}}-{{.Arch}}{{.OSArchiveExt}}
          os-mapping:
            darwin: macosx
          arch-mapping:
            arm64: aarch64
            amd64: x64
          ext-mapping:
            windows: -jdk.zip # if an extension other than the default is required
            linux: .tar.gz # if an extension other than the default is required
            darwin: .tar.gz # if an extension other than the default is required
      graal-17:
        id: graal
        vendor: Oracle
        type: Download
        endpoint: https://download.oracle.com/java/21/archive/jdk-21.0.1_windows-x64_bin.zip
  - name: maven
    display-name: Maven
    description: |
      Maven (3.9.6)                                          https://maven.apache.org/

      Apache Maven is a software project management and comprehension tool. Based on
      the concept of a project object model (POM), Maven can manage a project's build,
      reporting and documentation from a central piece of information.
    export-path: bin # if a subpath needs to be exported
    provider:
      maven:
        id: maven
        type: MavenRelease
        endpoint: org.apache.maven:apache-maven
        settings:
          archive: apache-maven-{{.Version}}-bin.tar.gz
  - name: groovy
    display-name: Groovy
    description: |
      Groovy (4.0.16)                                      http://www.groovy-lang.org/

      Groovy is a powerful, optionally typed and dynamic language, with static-typing
      and static compilation capabilities, for the Java platform aimed at multiplying
      developers' productivity thanks to a concise, familiar and easy to learn syntax.
      It integrates smoothly with any Java program, and immediately delivers to your
      application powerful features, including scripting capabilities, Domain-Specific
      Language authoring, runtime and compile-time meta-programming and functional
      programming.
    export-path: bin
    provider:
      groovy:
        id: groovy # if id is equal to the candidate name, it will be omitted
        type: MavenRelease
        endpoint: org.apache.maven:apache-maven
      groovy-legacy:
        id: groovy # if id is equal to the candidate name, it will be omitted
        type: MavenRelease
        endpoint: org.codehaus.groovy:groovy-binary
  - name: k9s
    description: |
      k9s (0.30.3)                                     https://github.com/derailed/k9s
      
      Kubernetes CLI To Manage Your Clusters In Style! 
      
      K9s provides a terminal UI to interact with your Kubernetes clusters. The aim of
      this project is to make it easier to navigate, observe and manage your applica-
      tions in the wild. K9s continually watches Kubernetes for changes and offers 
      subsequent commands to interact with your observed resources.
    provider:
      k9s-github:
        id: k9s
        vendor: derailed
        type: GitHubRelease
        endpoint: derailed/k9s # org-id/repo
        pre-releases: true #default: false
        ext-mapping:
          windows: zip # if an extension other than the default is required
          linux: tar.gz # if an extension other than the default is required
          darwin: tar.gz # if an extension other than the default is required
  - name: kubectl
    description: |
      kubectl
    provider:
      kubectl-github:
        type: GitHubTagsDownloadUrl
        endpoint: kubernetes/kubernetes # org-id/repo
        version-cleanup: ^[[:alpha:]]+\s #regex to clean up version string, currently only supported by github based providers
        settings:
          url: https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl{{.OSArchiveExt}} # {{.OSArchiveExt}} includes the `.` dot before the extension
          file-mode: "0750" # only valid if a single uncompressed file is loaded
          ext-mapping:
            windows: .exe # if an extension other than the default is required
            #linux:  # if an extension other than the default is required
            #darwin: # if an extension other than the default is required
#  - argocd-cli:
#  - stern:
#  - github-cli:
```
    

# todo

- github.com/mholt/archiver/v3
- https://api.azul.com/metadata/v1/docs/swagger
- https://api.azul.com/metadata/v1/docs/alt_openapi.json
- https://archive.apache.org/dist/groovy/
- https://archive.apache.org/dist/maven/maven-3/
- https://archive.apache.org/dist/maven/maven-4/


