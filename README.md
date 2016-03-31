A simple bot for Slack that parses a JUnit XML file and sends the results to a Slack channel. 

The bot expects two environment variables: **SLACK_TOKEN** (the api token for your bot integration) and **CHANNEL** (the name of the channel you want to publish to). Once defined, simply run the compiled *testcrier* executable. It will listen on port 8080.


To build, set GOPATH to the location where the repo was cloned, then run *make*

There is also a Dockerfile that will build a small Docker image using Alpine Linux. Run the Docker image like so:

docker run -d -e SLACK_TOKEN=<TOKEN> -e CHANNEL=bot-testing -v /etc/ssl/certs:/etc/ssl/certs:ro -p 8080:8080 testcrier

You'll need to mount your SSL certificates into the container. 

To send an XML file:

curl -X POST -F "uploadfile=@automated-tests.xml" http://192.168.99.100:8080/test/automated-tests

Optionally, specify a channel name with a query param

curl -X POST -F "uploadfile=@automated-tests.xml" http://192.168.99.100:8080/test/automated-tests?channel=test-result-channel 
