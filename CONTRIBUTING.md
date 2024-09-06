
# Contributing Guidelines

Thank you for your interest in contributing to this project! Please take a moment to review these guidelines to make the process smooth and productive.

## Pull Requests for Adding Support for a New Chain

If you are adding support for a new blockchain, follow these instructions carefully:

### 1. JSON File for Supported Chains

- **File Location**: The JSON file must be added to the `supported-chains` directory.
- **File Structure**: [Explained on the README](README.md#json-chain-config-file)

### 2. JSON File for Integration Tests

- **File Location**: Add a corresponding JSON file for integration tests to the `integration-tests/test-data` directory.
- **File Structure**: The integration test file should include test cases for all the methods added in the support file. An example structure is provided below:

  ```json
  {
      "cases":[
          {
              "name": "eth_dummyMethodWithBlockNumber",
              "reqBody": {
                  "method": "eth_dummyMethodWithBlockNumber",
                  "params": ["0x1234567890abcdef", "latest"],
                  "id": 1,
                  "jsonrpc": "2.0"
              }
          },
          {
              "name": "eth_anotherDummyMethodWithBlockNumber",
              "reqBody": {
                  "method": "eth_anotherDummyMethodWithBlockNumber",
                  "params": ["latest"],
                  "id": 2,
                  "jsonrpc": "2.0"
              }
          }
      ]
  }
  ```

- **Explanation of Fields**:
  - **`name`**: A descriptive name for the test case.
  - **`reqBody`**: The request body that will be sent to the method being tested.
    - **`method`**: The method name being tested.
    - **`params`**: An array of parameters to pass to the method, including block number or range if applicable.
    - **`id`**: A unique identifier for the request (can be any number).
    - **`jsonrpc`**: The JSON-RPC version, typically "2.0".

### 3. Running Integration Tests

After adding the necessary files, you can run the integration tests to ensure everything is working correctly.

- **Command to Run Tests**: You need to be in the `integration-tests` directory and run the following command:

  ```bash
  go test -v -url=https://dummy.blockchain.node -integration=true -keyfile=../rpc-context/test-data/.dummy_key.pem -configFile=../supported-chains/dummychain.json -integrationFile=test-data/dummy_integration_test.json -waitTime=500
  ```

- **Explanation of Flags**:
  - **`-v`**: Enables verbose mode, so you can see detailed output from the test execution.
  - **`-url`**: The URL of the blockchain node you are testing against. Replace with the appropriate node URL.
  - **`-integration`**: Set this flag to `true` to enable integration testing.
  - **`-keyfile`**: The path to the key file used for signing transactions during tests. Replace with the appropriate key file.
  - **`-configFile`**: The path to the JSON file in the `supported-chains` directory that specifies the methods for the new chain.
  - **`-integrationFile`**: The path to the integration test JSON file located in the `integration-tests/test-data` directory.
  - **`-waitTime=`**: Wait time in between requests in miliseconds. This flag is optional and it's used to prevent the tests to be rate limited.

### 4. Review Checklist

Before submitting your PR, make sure:

- [ ] The JSON file for the supported chain is correctly formatted and placed in the `supported-chains` directory.
- [ ] The corresponding integration test JSON file is added to the `integration-tests/test-data` directory.
- [ ] Both JSON files are properly structured as per the provided examples.
- [ ] Tests are passing, and any new methods added are adequately covered by the integration tests.

## Pull Requests for Other Features or Bug Fixes

If your PR is not related to adding support for a new chain:

- **Unit Tests**: Please ensure that you add or update unit tests to cover the new features or bug fixes you are implementing.
  - Make sure your changes are thoroughly tested and that the test coverage is adequate.
  - Place the new unit tests in the appropriate test file or create a new one if necessary.
- **Documentation**: Update any relevant documentation to reflect your changes.
- **Opening the PR**: You can simply open your PR without the need for additional files or integration tests as outlined for chain support.

### 5. Opening the Pull Request

When opening your PR:

- **Title**: Clearly describe the feature or fix you are contributing.
- **Description**: Include any relevant details about the changes made, how they were tested, and any potential impacts.
- **Link Issues**: If applicable, link any relevant issues.

## Additional Information

If you have any questions, feel free to reach out or open an issue for clarification.

Thank you for contributing!
