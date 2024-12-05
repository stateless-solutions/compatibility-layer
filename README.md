# Stateless Compatibility Layer

This repository contains a Dockerized Go application for the Stateless Compatibility Layer. This document provides instructions on how to set up and run the application using Docker.

## Prerequisites

Before you begin, ensure you have the following installed on your system:

- Docker
- Go (for development purposes)

## Setup

1. **Clone the repository:**

    ```sh
    git clone https://github.com/stateless-solutions/compatibility-layer.git
    cd stateless-compatibility-layer
    ```

2. **Environment Variables:**

    Create a `.env` file in the root directory of your project by copying the example file:

    ```sh
    cp .env.example .env
    ```

    Edit the `.env` file to set the necessary environment variables. The required variables are:

    ```ini
    DEFAULT_CHAIN_URL=http://your-chain.co
    USE_ATTESTATION=true
    KEY_FILE=/path/to/your/.key.pem
    KEY_FILE_PASSWORD=
    IDENTITY=identity
    HTTP_PORT=8080
    CONFIG_FILES=/path/to/your/chain.json
    LOG_LEVEL='0'
    GATEWAY_MODE=true
    ```

    `LOG_LEVEL` is an int representing the level of logging desired based on [slog's standard](https://cs.opensource.google/go/go/+/refs/tags/go1.23.2:src/log/slog/level.go;l=17) 
    Ensure that the `KEY_FILE` and `CONFIG_FILES` paths point to the actual locations of your files.
    `GATEWAY_MODE` is to change regular rpc to methods to their custom counterpart when a request is made

## Building the Docker Image

1. **Build the Docker image:**

    Navigate to the root directory of the project and run the following command:

    ```sh
    docker build -t comp-layer .
    ```

    This command builds the Docker image and tags it as `comp-layer`.

## Running the Application

1. **Ensure your key and configuration files are in place:**

    Make sure the key file specified in the `KEY_FILE` and `CONFIG_FILES` environment variable exist and are accessible.

    <a name="json-chain-config-file"></a> 
    **JSON chain config file:**

    There is an example of this file in `supported-chains/ethereum.json`

    - **File Structure**: Ensure that the JSON file adheres to the following structure:

    ```json
    {
        "chainNames": ["ethereum"],
        "chainType": "evm",
        "methods": [
            {
                "customMethod": "eth_dummyMethodWithBlockNumber",
                "originalMethod": "eth_dummyMethod",
                "positionsGetterParam": [1],
                "isRange": false
            },
            {
                "customMethod": "eth_anotherDummyMethodWithBlockNumber",
                "originalMethod": "eth_anotherDummyMethod",
                "positionsGetterParam": [0],
                "customHandler": "HandleAnotherDummyMethodWithBlockNumber",
                "isRange": true
            }
        ]
    }
    ```

    - **Explanation of Concepts**:
        - **`getterStuct`**: The type of the data that needs to be gotten to add to the stateless custom methods. This is a generic specified on `GetterStruct` in the `custom-rpc-methods/custom_rpc_methods.go` file. Current supported getter structs for each chain type are:
            - **`EVM`**: the geth's standard [BlockNumbeOrHash](https://github.com/ethereum/go-ethereum/blob/master/rpc/types.go#L146)
            - **`Solana`**: the solana go's standard [CommitmentType](https://github.com/gagliardetto/solana-go/blob/main/rpc/types.go#L431)

    - **Explanation of Fields**:
        - **`chainNames`**: Name of the chains of the config file, this field is just for trackability purposes it is not used in this repo
        - **`chainType`**: Chain type of the chain of the config file, this field is a enum specified on `ChainType` in the `custom-rpc-methods/custom_rpc_methods.go` file
        - **`customMethod`**: The method name used to retrieve data along with the getter struct (block number in the case of EVM).
        - **`originalMethod`**: The original method name without the getter struct support.
        - **`positionsGetterParam`**: Represents the 0-based index positions of the getter struct parameters (block number in the case of EVM) in the method's parameter list. If there are multiple positions specified (indicating a range), the first one is treated as the "from", and the last one as the "to". For this range to be used, the `isRange` parameter must be set to `true`; otherwise, any additional positions params will be ignored.
        For EVM chains the parameters at these positions must be of the geth's standard [BlockNumbeOrHash](https://github.com/ethereum/go-ethereum/blob/master/rpc/types.go#L146) type to extract the block number(s). If the parameters are of a different type, you should implement a custom handler to process them correctly.
        In case this field is empty it will be assumed all requests use the default block tag: latest for non range and earliest to latest for range.
        - **`customHandler`**: The name of the custom handler function that must be used for the method. 
        This handler must have the following signature for EVM chains: `func(*models.RPCReq) ([]*rpc.BlockNumberOrHash, error)`. It is mandatory that this function be implemented as a public method within the `evmCustomHandlersHolder` struct, which is located in the `custom-rpc-methods/evm_custom_handlers.go` file.
        This parameter is optional. If not specified, the method will default to using the `positionsGetterParam` to extract the getter struct(s).
        - **`isRange`**: A boolean indicating whether the method supports a range.

2. **Run the Docker container:**

    Use the following command to run the Docker container. This command reads environment variables from the `.env` file, mounts the key and configuration files, and starts the application:

    ```sh
    docker run --env-file=.env -v /path/to/your/.key.pem:/app/.key.pem -v /path/to/your/chain.json:/app/chain.json -d -p 8080:8080 --name comp-layer comp-layer --p=true
    ```

    - `--env-file=.env`: Loads the environment variables from the `.env` file.
    - `-v /path/to/your/.key.pem:/app/.key.pem`: Mounts the key file into the Docker container. Ensure this path matches the `KEY_FILE` variable in the `.env` file.
    - `-v /path/to/your/chain.json:/app/chain.json`: Mounts the config file into the Docker container. Ensure this path matches the `CONFIG_FILES` variable in the `.env` file.
    - `-d`: Runs the container in detached mode.
    - `-p 8080:8080`: Maps port 8080 on your host to port 8080 in the container. The exposed port depends on the `HTTP_PORT` environment variable.

## Accessing the Application

Once the container is up and running, the application will be accessible at `http://localhost:8080` (or the port specified in `HTTP_PORT`).

## Stopping the Application

To stop the running Docker container, use the following command:

```sh
docker stop comp-layer
```

## Removing the Container

To remove the stopped Docker container, use the following command:

```sh
docker rm comp-layer
```

## Notes

- Ensure the `DEFAULT_CHAIN_URL`, `KEY_FILE`, `KEY_FILE_PASSWORD`, `IDENTITY`, `CONFIG_FILES` and `HTTP_PORT` environment variables are correctly set in your `.env` file.
- The `CONFIG_FILES` env var supports multiple files, just input them separated by a `,` with no spaces.
- The port specified in the `HTTP_PORT` environment variable should match the port mapping in the Docker run command.
- The chain URL used for a request can be specified in a header called `Stateless-Chain-URL`. If this header is present in a request, its value will take precedence over any URL set in the environment variable.
- If the `USE_ATTESTATION` is set to true, the `KEY_FILE` and `IDENTITY` env vars are mandatory for the attestations to work.

## Troubleshooting

- **Port Conflicts**: If the specified port is already in use, you can change the `HTTP_PORT` variable in the `.env` file and update the port mapping in the Docker run command accordingly.
- **Key File and Config Files**: Ensure the file paths are correct and the files have the necessary permissions.

For further assistance, please contact the repository maintainer.

## Getting Help and Customization

This README should help you get started with running the Stateless Compatibility Layer using Docker. The Docker setup is designed to be flexible so you can deploy it however you want. Feel free to create your own Docker Compose file, Makefile, or other deployment configurations. If you encounter any issues or have questions, feel free to open an issue in the repository.
