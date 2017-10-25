# Simple diagram service

Simple diagram service based on [mermaid](https://github.com/knsv/mermaid) with in memory LRU cache. It generates diagrams to be embedded in documents.

## Quick Start

- Install [NodeJs](https://nodejs.org/en/download/current/)
- Install [Go](https://golang.org/dl/)
- Run following commands

   ```bash
   $ npm install -g mermaid.cli
   $ go get -d github.com/ng-vu/mermaid-service
   $ go install github.com/ng-vu/mermaid-service
   $ $GOPATH/bin/mermaid-service
   ```

- Go to http://localhost:8080
- Learn more about the syntax on [mermaidjs.github.io](https://mermaidjs.github.io/)

# License

- [MIT License](https://opensource.org/licenses/mit-license.php)
