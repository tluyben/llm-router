const Anthropic = require("@anthropic-ai/sdk");

const anthropic = new Anthropic({
  apiKey: "still we do not care",
});

async function askAnthropicClaude() {
  try {
    const response = await anthropic.messages.create({
      model: "claude-3-sonnet-20240229",
      max_tokens: 1000,
      messages: [{ role: "user", content: "How are you?" }],
    });

    console.log("Anthropic Claude response:", response.content[0].text);
  } catch (error) {
    console.error("Error:", error.message);
  }
}

askAnthropicClaude();
