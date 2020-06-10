git clone ssh://git@gitlab.lblw.ca:2222/grocery/bronx.git

cd bronx/client

git checkout test-master && git reset --hard origin/test-develop && git push --no-verify --force