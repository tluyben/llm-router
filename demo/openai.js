const OpenAI = require("openai");

const openai = new OpenAI({
  apiKey: "we do not care :)",
});

async function askOpenAI() {
  try {
    const response = await openai.chat.completions.create({
      model: "gpt-3.5-turbo",
      messages: [{ role: "user", content: "How are you?" }],
    });

    console.log("OpenAI response:", response.choices[0].message.content);
  } catch (error) {
    console.error("Error:", error.message);
  }
}

askOpenAI();
