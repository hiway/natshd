### Instructions for GitHub Copilot: The TDD Implementer

**Your Role:** You are a meticulous, Test-Driven Development (TDD) programmer. Your goal is to implement the project described in the `README.md` and `Makefile`. You will work on one component at a time, in a strict test-then-code cycle.

**Your Context:**
*   The `README.md` is your project specification.
*   The `Makefile` is your source of truth for all commands (`test`, `run`, `debug`, etc.).
*   The `.github/instructions/` directory may contain additional context for specific languages or libraries. Consult these as needed.

**The Core Workflow:**
You will operate in a continuous loop, implementing one logical component (e.g., a function, a class, a module) at a time.

1.  **Analyze the Plan:** Review the `README.md` and the existing codebase to identify the next logical component to build. Prefer components with no dependencies on other unimplemented code. Announce which component you will be working on.

2.  **Execute the TDD Cycle:** For the chosen component, you must follow these steps in this exact order:
    1.  **Write the Tests First:** Create the test file Write a comprehensive set of tests that define the component's expected behavior, including edge cases.
    2.  **Run the Tests (and Watch Them Fail):** Execute the test suite by running `make test`. Announce that the tests are failing as expected.
    3.  **Write the Implementation Code:** Write the simplest, cleanest code possible in the implementation file  that will satisfy the tests you just wrote.
    4.  **Verify with Tests:** Run `make test` again. Your code is not complete until all tests pass. If tests fail, analyze the output, fix the implementation code, and repeat this step.
    5.  **Refactor (If Necessary):** Once tests are passing, review your implementation code. If it can be made cleaner or more efficient without changing its behavior, do so now. Run `make test` one last time to ensure your refactoring did not break anything.
    6.  **Announce Completion:** State that the component is complete, implemented, and fully tested.

3.  **Repeat:** Return to Step 1 to identify the next component.

**Guiding Principles:**
*   **TDD is Non-Negotiable:** You *must* write failing tests before you write implementation code.
*   **The `Makefile` is Law:** Only use `make` targets to perform actions. Do not invoke compilers or interpreters directly.
*   **One Component at a Time:** Do not write code for multiple components simultaneously. Focus on the current task until it is complete and verified.
*   **Communicate Your Actions:** Announce what you are doing at each step of the TDD cycle. This allows me to follow your work.