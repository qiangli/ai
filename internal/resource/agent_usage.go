package resource

// describe each agent in 18 words or less
var AgentDesc = map[string]string{
	"ask":  "A versatile Q&A tool providing concise, reliable answers on diverse topics for learners and inquisitive minds",
	"seek": "Internet agent for gathering insights and information, aiding research, analysis, and staying informed with timely, relevant content",
	"sql":  "An SQL agent that simplifies query generation, boosts productivity, and assists users from beginners to database administrators",
	"eval": "Direct AI interaction tool for creative brainstorming, rapid prototyping, and spontaneous problem-solving without predefined workflows",
	"oh":   "OpenHands aids incremental development for greenfield projects, code integration, refactoring, and debugging with context.",
	"gptr": "GPT Researcher autonomously conducts thorough web research, generating factual reports with citations",
	"git":  "A Git agent for generating git messages in software development projects",
	"code": "Code agent enables collaborative code editing with LLMs in your local workspace",
}

// summarize each agent in 100 words
var AgentInfo = map[string]string{
	"ask": `
	This agent serves as an all-encompassing Q&A platform, enabling users to explore and inquire about a diverse array of topics, from scientific phenomena to cultural practices and technology. Its primary objective is to deliver reliable, relevant, and accessible answers that are both informative and concise. Ideal for learners, educators, and the inquisitive, it offers an invaluable tool for broadening knowledge and swiftly addressing everyday questions. By providing clear and explanatory answers, this agent supports users in their quest for understanding across a wide range of subjects.
`,
	"seek": `
	This agent serves users seeking to gather insights from the internet by using advanced search technologies. It acts as a gateway to deliver comprehensive, up-to-date information from diverse online sources. Whether for academic research, competitive business analysis, or staying informed on trends and news, this tool efficiently supports digital exploration. It skillfully navigates the vast landscape of online resources to provide accurate, timely, and contextually relevant information. Ideal for users needing a reliable means of accessing and analyzing digital content, it ensures that pertinent information is readily available at their fingertips.
`,
	"sql": `
	This SQL agent streamlines query generation, enabling users to derive insights from databases by asking questions without requiring SQL expertise. It integrates with databases like PostgreSQL, ensuring security and efficiency through privacy and improved accuracy with continuous training. Designed for both enthusiasts and professionals, the tool assists in writing, modifying, and optimizing SQL queries, offering guidance on a wide range of tasks from schema management to complex query execution. By simplifying SQL interactions, it accelerates query development, aids debugging, and boosts productivity, serving both beginners and expert database administrators alike.
`,
	"eval": `
	This tool enables direct, conversational interaction with AI, eliminating traditional prompts or predefined workflows. It offers a dynamic interface for users to test ideas, hypothesize, or simply engage with AI in a fluid, natural way without system constraints. Particularly valuable for creative brainstorming and rapid prototyping, it allows for quick, unstructured advice. By offering a flexible interaction model, users can explore diverse scenarios and fully utilize the AI's capabilities for spontaneous problem-solving, experimentation, and innovation across various fields. This promotes a more adaptable and creative approach to AI engagement.
`,
	"oh": `
    OpenHands is an engineering assistant tool that simplifies tasks by encouraging an incremental approach. Start with basic exercises like creating a "hello world" script and progressively improve your project. It's ideal for greenfield projects, allowing you to begin with simple tasks such as developing a React TODO app and gradually adding features. OpenHands effectively integrates new code into existing systems and supports step-by-step code refactoring. For troubleshooting, detailed context is essential. The best outcomes are achieved by approaching tasks in small steps, providing specific details, sharing context, and making frequent commits.
`,
	"gptr": `
	The GPT Researcher is an autonomous agent designed for thorough web research, producing detailed, factual reports with citations. It efficiently generates research questions and gathers information, addressing misinformation and token limitations common in language models. This tool offers extensive customization and aggregates data from over 20 sources, allowing for report export in multiple formats. Its user-friendly frontend enhances interaction and real-time progress tracking, making research tasks more precise and manageable for individuals and organizations seeking accurate information.
`,
	"git": `
    A Git agent is a tool used in software development to aid in creating and managing Git commit messages. It automates and enhances the process, ensuring consistency and clarity in the project's change history. By producing well-structured commit messages, a Git agent helps maintain organized records, facilitates team collaboration, and improves overall efficiency in version control management.
`,
	"code": `
	The Code agent is an innovative tool that boosts collaborative software development by seamlessly integrating Large Language Models (LLMs) into local environments. By harnessing the power of these models, it can generate new code, refactor existing code, fix bugs, and develop test cases efficiently. This enhances productivity and code quality, making teamwork more efficient and effective.
`,
}
