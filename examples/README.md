# Examples

Want to run an example? Follow the instructions below and change the commands based on which example you want to run, etc.

## Download an Example

You can copy/paste the examples in any of these subfolders into a local file. You can also just use [curl](https://curl.se/docs/tutorial.html):

```bash
# Download an example. Replace /tables/ with whatever example you want to download.
curl -L https://raw.githubusercontent.com/datastax/astra-db-go/main/examples/tables/main.go -o main.go
```

## Initialize a go module and get dependencies

Next, initialize a go module using [`go mod init`](https://go.dev/doc/tutorial/create-module) and get dependencies (you can use [`go mod tidy`](https://go.dev/ref/mod#go-mod-tidy)):

```bash
# Initialize a go module
go mod init github.com/example/module
# Get dependencies
go mod tidy
```

## Configure the example with your database endpoint / token

Create a `.env` file with the following values:

```.env
ASTRA_DB_API_ENDPOINT="https://my-database-id-my-region.apps.astra.datastax.com"
ASTRA_DB_APPLICATION_TOKEN="AstraCS:myToken"
```

If you have the [Astra CLI](https://docs.datastax.com/en/astra-cli/index.html) installed, you can have it create the .env file for you:

```bash
# Replace mydb with the name of your database
astra db create-dotenv mydb
```

## Run the example

You can now run the example and interact with your database:

```bash
# Run the example
go run .
```