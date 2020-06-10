golang env setup

1. download and install golang: https://golang.org/doc/install?download=go1.14.4.darwin-amd64.pkg

2. export $GOPATH in .bash_profile (export GOPATH=$HOME/go)

3. in path $HOME/go, create folder pkg, bin, src

4. clone the project into src folder

5. in project folder, run command: go install

run script

1. before running script, make sure develop branch is up-to-date for release

2. run command: go build

3. run command:
   
   GITLAB_TOKEN=`<personal token>` ./bronx.release


TODOs:

1. add .gitlab-ci.yaml

2. replace personal token with global token

3. adjust tag title, message and release note (not accurate now)

4. add error handling and better logging

5. need to delete generated bronx folder if running script on local