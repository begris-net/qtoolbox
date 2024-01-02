# qtoolbox
Simple SDK and tool helper

# Config

```yaml
update_url: https://qsdk.github.io
candidates:
  - java:
      - zulu:
          id: zulu
          type: Zulu
          endpoint: https://api.azul.com/metadata/v1          
      - amazon:
          id: amzn
          type: Zulu
          endpoint: https://api.azul.com/metadata/v1
      - graal:
  - maven:
      - maven-3:
          id: maven # if id is equal to the candidate name, it will be omitted
          type: ApacheArchive          
          endpoint: https://archive.apache.org/dist/maven/maven-3
      - maven-4:
          id: maven
          type: ApacheArchive
          endpoint: https://archive.apache.org/dist/maven/maven-4
  - groovy:
      - groovy:
          id: groovy # if id is equal to the candidate name, it will be omitted
          type: ApacheArchive
          endpoint: https://archive.apache.org/dist/groovy
  - k9s:
      - k9s-github:
          id: k9s
          type: GitHubRelease
          endpoint: derailed/k9s # org-id/repo
  - kubectl:
  - argocd-cli:
  - stern:
  - github-cli:
  - 
  

```
    

# todo

- github.com/mholt/archiver/v3
- https://api.azul.com/metadata/v1/docs/swagger
- https://api.azul.com/metadata/v1/docs/alt_openapi.json
- https://archive.apache.org/dist/groovy/
- https://archive.apache.org/dist/maven/maven-3/
- https://archive.apache.org/dist/maven/maven-4/


