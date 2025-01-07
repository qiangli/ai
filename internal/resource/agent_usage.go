package resource

const MiscCommandUsage = `
list                    List available binaries in the path
info                    Show system information
`

var AgentList = map[string]string{
	"ask":  "Ask general questions on a broad range of topics",
	"seek": "Explore the web for up to date information",
	"sql":  "Generate SQL queries for your dataset",
	"eval": "Send direct messages without system prompts for evaluation",
}

var AgentInfo = map[string]string{
	"ask": `
	This agent allows users to pose general questions that cover a broad range of topics.
Whether you're curious about scientific phenomena, historical events, cultural practices,
or even the intricacies of technology, this option empowers you to seek explanatory answers.
It acts as a digital Q&A platform, where the primary goal is to provide reliable, relevant,
and accessible answers that are informative and concise. Ideal for learners, educators,
and inquisitive minds, this capability serves as an invaluable tool for expanding knowledge
and providing quick solutions to everyday questions.
`,
	"seek": `
	This agent is designed for users who want to traverse the internet to gather information
and insights. It acts as a gateway to the World Wide Web, harnessing the power of search technologies
to deliver comprehensive and up-to-date information from a multitude of online sources. Whether you're
conducting research for academic purposes, performing competitive analysis in business,
or simply looking to stay informed about the latest trends and news, this feature equips you
with the necessary tools to efficiently explore digital content.
It adeptly navigates the vastness of online resources to ensure accurate, timely,
and contextually relevant information is at your fingertips.`,
	"sql": `
	This SQL agent is tailored for database enthusiasts and professionals who require assistance with SQL queries.
This tool is essential for individuals seeking to write, modify, or optimize database queries,
providing valuable guidance and suggestions. It caters to a wide range of SQL-related tasks,
from managing database schemas to executing complex queries and analyzing their results.
By simplifying the SQL interaction process, this assistant accelerates query development,
aids in debugging, and enhances overall database productivity. Whether you're a beginner
looking to understand the basics or an experienced database administrator aiming for optimization,
this tool offers indispensable support.
`,
	"eval": `
	This tool allows users to engage directly with AI, bypassing traditional prompts or predefined workflows.
It creates a conversational interface where users can test ideas, hypothesize, or simply interact with an AI
in a more fluid and natural manner without the constraints of system-imposed prompts.
This mode is particularly useful for creative brainstorming, rapid prototyping of ideas,
or when you need quick, unstructured advice.
By providing a more flexible interaction model, users can explore various scenarios
or harness the AI's capabilities for off-the-cuff problem-solving, experimentation, and innovation in any field of interest.
`,
}
