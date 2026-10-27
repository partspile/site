package ui

import (
	"github.com/partspile/site/internal/config"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const legalEffectiveDate = "July 27, 2026"

func legalSection(title string, body ...g.Node) g.Node {
	return legalSectionID("", title, body...)
}

func legalSectionID(id, title string, body ...g.Node) g.Node {
	attrs := []g.Node{Class("space-y-3 scroll-mt-20")}
	if id != "" {
		attrs = append(attrs, ID(id))
	}
	attrs = append(attrs,
		H2(
			Class("text-xl font-semibold text-zinc-900 dark:text-zinc-100"),
			g.Text(title),
		),
		Div(Class("space-y-3"), g.Group(body)),
	)
	return Div(attrs...)
}

func legalP(children ...g.Node) g.Node {
	attrs := []g.Node{
		Class("text-base leading-relaxed text-zinc-700 dark:text-zinc-300"),
	}
	attrs = append(attrs, children...)
	return P(attrs...)
}

func legalList(items ...g.Node) g.Node {
	return Ul(
		Class("list-disc pl-5 space-y-2 text-base leading-relaxed "+
			"text-zinc-700 dark:text-zinc-300"),
		g.Group(items),
	)
}

func legalLi(children ...g.Node) g.Node {
	return Li(g.Group(children))
}

func legalIntro(kind string) []g.Node {
	return []g.Node{
		pageTitle(kind),
		P(
			Class("mt-2 text-sm text-zinc-500 dark:text-zinc-400"),
			g.Text("Effective "+legalEffectiveDate),
		),
	}
}

func PrivacyPolicyPage() []g.Node {
	name := config.ServerName
	host := config.PublicHost()
	nodes := legalIntro("Privacy Policy")
	nodes = append(nodes,
		Div(
			Class("mt-10 space-y-10"),
			legalSection("Who we are",
				legalP(
					g.Textf("%s (“we”) operates %s, a "+
						"classifieds marketplace. This policy "+
						"explains how we handle information when "+
						"you use the site.", name, host),
				),
			),
			legalSection("Information we collect",
				legalP(g.Text("You provide:")),
				legalList(
					legalLi(g.Text(
						"Account details: username, password, phone "+
							"number, and optional account photo")),
					legalLi(g.Text(
						"Ads: title, description, category details "+
							"(such as price), location, and photos")),
					legalLi(g.Text(
						"Messages you send other users about ads")),
					legalLi(g.Text(
						"Anything you send us for support or account "+
							"recovery")),
				),
				legalP(g.Text("We also collect automatically:")),
				legalList(
					legalLi(g.Text(
						"Session and security cookies (login, CSRF)")),
					legalLi(g.Text(
						"Preference cookies (search filters, timezone, "+
							"distance unit, view mode)")),
					legalLi(g.Text(
						"Server logs needed to run and secure the "+
							"service (such as IP address, user agent, "+
							"and timestamps)")),
					legalLi(g.Text(
						"In-product activity we store (bookmarks and "+
							"ad or image click counts)")),
				),
			),
			legalSection("How we use it",
				legalList(
					legalLi(g.Text(
						"Create and secure your account (phone "+
							"verification and password recovery)")),
					legalLi(g.Text(
						"Publish and display your ads and images")),
					legalLi(g.Text(
						"Enable messaging between buyers and sellers")),
					legalLi(g.Text(
						"Send SMS notifications about unread messages "+
							"(you can turn these off)")),
					legalLi(g.Text(
						"Operate search and related site features")),
					legalLi(g.Text(
						"Prevent abuse, enforce our Terms, and comply "+
							"with law")),
				),
			),
			legalSection("How we store and protect it",
				legalP(g.Text(
					"We use third-party hosting and storage providers "+
						"to run the site and keep ads, images, and "+
						"account data. Passwords are hashed. Sensitive "+
						"fields such as name and phone are encrypted "+
						"at rest. We use reasonable safeguards, but no "+
						"method of transmission or storage is "+
						"completely secure.",
				)),
			),
			legalSection("Who we share with",
				legalList(
					legalLi(g.Text(
						"Service providers that help us run the site "+
							"(hosting, SMS delivery, storage), only as "+
							"needed to provide the service")),
					legalLi(g.Text(
						"Other users see what you publish: ads, photos, "+
							"and messages they are part of")),
					legalLi(g.Text(
						"Authorities when required by law, or to "+
							"protect rights and safety")),
				),
				legalP(g.Text(
					"We do not sell your personal information. We do "+
						"not share your phone number with other users "+
						"or use it for marketing unrelated to your "+
						"account.",
				)),
			),
			legalSection("SMS",
				legalP(g.Text(
					"By verifying your phone and using messaging "+
						"features, you agree we may text you about "+
						"account security and message notifications. "+
						"Message and data rates may apply. You can "+
						"turn notifications off in ",
				), faqLink("/auth/user/settings", "Settings"),
					g.Text(", or reply STOP to a text from us. "+
						"See also "),
					faqLink("/faq/sms-notifications",
						"SMS notifications FAQ"),
					g.Text("."),
				),
			),
			legalSection("Cookies",
				legalP(g.Text(
					"We use essential cookies for login and security, "+
						"and functional cookies for preferences. "+
						"Disabling cookies may break login or other "+
						"features.",
				)),
			),
			legalSection("Retention and deletion",
				legalP(g.Text(
					"You can delete your account in Settings. Your ads "+
						"are permanently deleted. Your username cannot "+
						"be reused. Your phone number is held for about "+
						"10 days before it can be registered again.",
				)),
				legalP(g.Text(
					"Soft-deleted records and backups may remain for a "+
						"limited time for security, abuse prevention, "+
						"and recovery. We may keep information longer "+
						"when required by law or for disputes.",
				)),
			),
			legalSection("Your choices",
				legalList(
					legalLi(
						g.Text("View and edit your profile, ads, and "+
							"notification settings in "),
						faqLink("/auth/user/settings", "Settings"),
					),
					legalLi(
						g.Text("Delete your account anytime in "),
						faqLink("/auth/user/settings", "Settings"),
					),
					legalLi(
						g.Text("Report problem ads by throwing a rock. "+
							"See "),
						faqLink("/faq/rocks", "What are rocks for?"),
					),
				),
			),
			legalSection("Children",
				legalP(
					g.Textf(
						"%s is not directed to children under 13, and "+
							"we do not knowingly collect personal "+
							"information from them. If we learn that we "+
							"have collected information from a child "+
							"under 13, we will delete the account and "+
							"related data. Minimum age to use the "+
							"service is in our ",
						name,
					),
					faqLink("/terms", "Terms of Service"),
					g.Text("."),
				),
			),
			legalSection("Content reports",
				legalP(
					g.Text("Listing and content problems are handled "+
						"in-product: throw a rock on the ad. See "),
					faqLink("/faq/rocks", "What are rocks for?"),
					g.Text(" and our "),
					faqLink("/terms", "Terms of Service"),
					g.Text("."),
				),
			),
			legalSection("Changes",
				legalP(g.Text(
					"We may update this policy. The effective date at "+
						"the top will change when we do. Continued use "+
						"of the site after an update means you accept "+
						"the revised policy.",
				)),
			),
		),
	)
	return nodes
}

func TermsOfServicePage() []g.Node {
	name := config.ServerName
	email := config.ContactEmail
	host := config.PublicHost()
	nodes := legalIntro("Terms of Service")
	nodes = append(nodes,
		Div(
			Class("mt-10 space-y-10"),
			legalSection("Agreement",
				legalP(
					g.Textf("By using %s you agree to these "+
						"Terms and our ", host),
					faqLink("/privacy", "Privacy Policy"),
					g.Text(". If you do not agree, do not use the "+
						"service."),
				),
			),
			legalSection("Eligibility",
				legalP(g.Textf(
					"You must be at least 18 years old and able to "+
						"form a binding contract. You need a phone "+
						"number you control to register and use the "+
						"service. One account per person, and one "+
						"active account per phone number. Do not "+
						"register with someone else’s number or "+
						"impersonate another person. %s is for "+
						"personal use as a classifieds marketplace.",
					name,
				)),
			),
			legalSection("Accounts",
				legalP(g.Text(
					"You are responsible for your password and for "+
						"activity on your account. If you think someone "+
						"else has access, change your password in "+
						"Settings (or recover the account with your "+
						"phone). We may suspend or terminate accounts "+
						"that violate these Terms or pose a risk to the "+
						"service or other users.",
				)),
			),
			legalSection("The service",
				legalP(g.Textf(
					"%s is a platform for posting and browsing "+
						"classifieds and messaging about listings. We "+
						"are not a party to transactions between users. "+
						"We do not guarantee that listings are accurate, "+
						"legal, or that users will complete deals.",
					name,
				)),
			),
			legalSection("Your content",
				legalP(g.Text(
					"You retain ownership of ads, images, and messages "+
						"you upload. You grant us a worldwide, "+
						"non-exclusive license to host, store, display, "+
						"reproduce, and process that content as needed "+
						"to operate the service (including thumbnails, "+
						"search, and moderation).",
				)),
				legalP(g.Text(
					"You represent that you have all rights needed to "+
						"post the content (including photos) and that "+
						"it does not infringe others’ rights. Public "+
						"ads and images may be visible to anyone. "+
						"Messages are visible to the people in the "+
						"conversation and may be processed to deliver "+
						"notifications.",
				)),
			),
			legalSectionID("prohibited", "Prohibited content and conduct",
				legalP(g.Text("You may not post or use the service for:")),
				legalList(
					legalLi(g.Text(
						"Illegal goods or services, fraud, or scams")),
					legalLi(g.Text(
						"Harassment, threats, or abuse of other users")),
					legalLi(g.Text(
						"Malware, scraping, or other technical abuse")),
					legalLi(g.Text("Impersonation or spam")),
					legalLi(g.Text(
						"Nonconsensual intimate images, including "+
							"deepfakes")),
					legalLi(g.Text(
						"Child sexual exploitation material")),
					legalLi(g.Text(
						"Content that violates others’ intellectual "+
							"property or privacy")),
					legalLi(g.Text(
						"Anything else that violates law or rules we "+
							"publish")),
				),
				legalP(g.Text(
					"We may remove content, restrict features, or ban "+
						"accounts when we believe these Terms or the "+
						"law require it.",
				)),
			),
			legalSection("Reporting and removal",
				legalP(
					g.Textf(
						"%s is self-policed with rocks. If an ad "+
							"violates these Terms or has other "+
							"problems, throw a rock on it. That starts "+
							"a conversation with the ad owner so you "+
							"can work it out. See ",
						name,
					),
					faqLink("/faq/rocks", "What are rocks for?"),
					g.Text("."),
				),
				legalList(
					legalLi(g.Text(
						"Rocks on an ad are visible on the listing; "+
							"anyone can open the dispute assessment "+
							"(parties labeled only as Owner and "+
							"Inquirer)")),
					legalLi(g.Textf(
						"An ad with more than %d rocks is excluded "+
							"from search listings",
						config.MaxRockCount,
					)),
					legalLi(g.Text(
						"Throwing requires choosing a reason; you may "+
							"review a provisional assessment first")),
					legalLi(g.Text(
						"When the issue is resolved, the thrower can "+
							"unthrow and reclaim the rock")),
				),
				legalP(g.Text(
					"Do not email us to report listing problems, "+
						"copyright claims, or other content issues — "+
						"use rocks. We may still remove content or "+
						"restrict accounts when needed to protect the "+
						"service or comply with law.",
				)),
			),
			legalSection("SMS",
				legalP(
					g.Text("Phone verification and message alerts are "+
						"part of the service. Consent, rates, and "+
						"opt-out are described in the "),
					faqLink("/privacy", "Privacy Policy"),
					g.Text(". You must use a phone number you control."),
				),
			),
			legalSection("Our intellectual property",
				legalP(g.Textf(
					"Site design, branding, software, and non-user "+
						"content belong to %s or our licensors. You may "+
						"not copy or reverse engineer the service "+
						"except as allowed by law.",
					name,
				)),
			),
			legalSection("Disclaimers",
				legalP(g.Text(
					"The service is provided “AS IS.” To the fullest "+
						"extent permitted by law, we disclaim "+
						"warranties of merchantability, fitness for a "+
						"particular purpose, and non-infringement. We "+
						"do not warrant uninterrupted or error-free "+
						"service.",
				)),
			),
			legalSection("Limitation of liability",
				legalP(g.Textf(
					"To the fullest extent permitted by law, %s’ total "+
						"liability for claims arising from the service "+
						"is limited to one hundred US dollars ($100). "+
						"We are not liable for indirect, incidental, "+
						"or consequential damages, or for disputes "+
						"between users. Some jurisdictions do not "+
						"allow these limits.",
					name,
				)),
			),
			legalSection("Indemnity",
				legalP(g.Textf(
					"You agree to defend and indemnify %s against "+
						"claims arising from your content, your use of "+
						"the service, or your violation of these Terms "+
						"or the law.",
					name,
				)),
			),
			legalSection("Termination",
				legalP(g.Text(
					"You may delete your account in Settings. We may "+
						"terminate or suspend access for breach or "+
						"risk. Provisions that should survive "+
						"(including licenses needed during wind-down, "+
						"disclaimers, and liability limits) survive "+
						"termination.",
				)),
			),
			legalSection("Governing law",
				legalP(g.Text(
					"These Terms are governed by the laws of the State "+
						"of Oregon, USA, without regard to "+
						"conflict-of-law rules.",
				)),
			),
			legalSection("Changes",
				legalP(g.Text(
					"We may update these Terms. The effective date at "+
						"the top will change when we do. Continued use "+
						"after an update means you accept the revised "+
						"Terms.",
				)),
			),
			legalSection("Contact",
				legalP(
					g.Text("Questions about these Terms: "),
					faqLink("mailto:"+email, email),
					g.Text("."),
				),
			),
		),
	)
	return nodes
}
