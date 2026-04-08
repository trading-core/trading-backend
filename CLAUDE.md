# Trading Backend
A mono repo which houses microservices and scripts in the cmd.

## Code Styleguide
- Do not abbreviate variables for the receiver, methods and structs. The only exception is context (ctx)
- If there are more than 3 parameters for the function input, please create a struct for the input, and copy the function/method name with the `Input` suffix. Keep the context as the first parameter and separate from the input.
- When writing new features, please write unit tests for it, particularly for abstract interfaces/implementations. Please ensure the test file package name has `_test`, and use `GoConvey` for the test framework. Use Behaviour Driven Development (BDD) methodology when writing test assertions.
- If the error is not client/validation side and is not expected at all within server side, use `fatal.OnError(err)`.