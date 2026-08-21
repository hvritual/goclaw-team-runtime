import { useState, type ComponentProps } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

type BaseInputProps = ComponentProps<"input"> & {
  onValueChange?: (value: string, eventDetails: { event: Event }) => void;
};

// Keep the public Input compatible with React's controlled onChange contract
// if its native implementation is ever replaced with Base UI again.
vi.mock("@base-ui/react/input", () => ({
  Input: ({ onChange: _onChange, onValueChange, ...props }: BaseInputProps) => (
    <input
      {...props}
      onChange={(event) =>
        onValueChange?.(event.target.value, { event: event.nativeEvent })
      }
    />
  ),
}));

import { Input } from "@multica/ui/components/ui/input";

function ControlledEmailInput() {
  const [email, setEmail] = useState("");

  return (
    <>
      <Input
        aria-label="邮箱"
        value={email}
        onChange={(event) => setEmail(event.target.value)}
      />
      <output data-testid="email-value">{email}</output>
    </>
  );
}

describe("Input", () => {
  it("supports controlled values through the React onChange API", async () => {
    const user = userEvent.setup();
    render(<ControlledEmailInput />);

    await user.type(screen.getByRole("textbox", { name: "邮箱" }), "tester@example.com");

    expect(screen.getByTestId("email-value")).toHaveTextContent("tester@example.com");
  });
});
