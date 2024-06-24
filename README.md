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
Simple SDK and tool manager.

# Installation

```shell
curl https://raw.githubusercontent.com/begris-net/qtoolbox/develop/install.sh | bash
```

To force a fresh creation of the qtoolbox directory use the ```--force``` argument.
```shell
curl https://raw.githubusercontent.com/begris-net/qtoolbox/develop/install.sh | bash -s -- --force
```


# Tools
## Supported so far
- Java JDK
  - Azul Zulu
  - Amazon Corretto
- Apache Maven
- Apache Groovy
- kubectl
- k9s
- stern
- OKD (OpenShift) client

## Tools planned
- ArgoCD Cli
- GitHub Cli
- ...

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
_TBD_ - description of repository metadata format 


# Local build command
```shell
for os in linux windows; do echo "Build $os version..."; GOOS=$os go build; done; ./qtoolbox setup --force
```
