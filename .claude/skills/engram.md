---
name: engram
description: Persistent AI memory companion. Use when the user shares information worth remembering, asks you to recall something, or when context from prior conversations would improve your response. Trigger on "remember", "remind me", "what do I know about", "capture this", brain dumps, meeting notes, or when the user shares facts about people, places, dates, or decisions.
---

# Engram — Persistent Memory Companion

## Overview

You have access to Engram, a persistent memory system. Use it proactively to capture important information and retrieve relevant context. The user should not have to ask you to remember things — if something is worth remembering, capture it.

## When to Capture

Save a thought whenever the user shares:
- **Decisions and rationale** — "We're going with Postgres because..." → capture the decision AND the why
- **People and relationships** — names, roles, how they met, impressions. Use `add_professional_contact` for professional contacts, `capture_thought` for informal mentions
- **Ideas and plans** — even half-formed ones. Tag as `idea` so they surface later
- **Tasks and commitments** — "I need to..." or "Remind me to..." → capture with action_items in metadata
- **Facts about their world** — addresses, account details, preferences, household info → use the appropriate structured tool (household items, maintenance tasks, recipes, etc.)
- **Meeting notes and conversations** — key takeaways, action items, who said what
- **Observations and reflections** — personal insights, lessons learned, patterns noticed

### Capture Rules

1. **Write standalone thoughts.** Each captured thought should make sense on its own when retrieved months later. Include enough context that future-you understands it without the conversation.
2. **Don't capture ephemeral debugging.** "This test is failing because of a typo" is not worth remembering. "The billing service silently drops requests over 1MB" is.
3. **Prefer structured tools when they fit.** A plumber's contact info → `add_vendor`. A recipe → `add_recipe`. A doctor appointment → `add_activity`. Fall back to `capture_thought` for anything that doesn't fit a structured tool.
4. **Capture the user's words, not your interpretation.** Paraphrase for clarity but preserve intent. Don't editorialize.
5. **One idea per thought.** If the user shares three things, capture three thoughts. This makes retrieval precise.

## When to Retrieve

Search Engram whenever:
- The user asks about something they've mentioned before — `search_thoughts` with the topic
- You're about to give advice and prior context would help — check what they've already thought about this
- The user mentions a person — `search_contacts` or `search_thoughts` with their name
- The user is planning and prior decisions are relevant — search for the topic area
- The user explicitly asks "what do I know about..." or "have I thought about..."

### Retrieval Rules

1. **Search before assuming.** If the user mentions a topic they might have captured before, search first. It takes one call and prevents you from giving context-free advice.
2. **Use the right search tool.** `search_thoughts` for ideas and general memory. `search_contacts` for people. `search_household_items` for home stuff. `search_recipes` for food. `get_upcoming_maintenance` for home tasks due.
3. **Lower the threshold for broad searches.** Default similarity is 0.5. For exploratory searches ("what have I said about leadership?"), drop to 0.3 to cast a wider net.
4. **Surface connections.** If a search returns thoughts that relate to each other or to the current conversation in non-obvious ways, point that out. Cross-pollination is one of the main values of persistent memory.

## Tool Selection Guide

### Core Memory
| Need | Tool |
|------|------|
| Save a general thought, idea, observation | `capture_thought` |
| Find thoughts by meaning | `search_thoughts` |
| Browse recent thoughts | `list_thoughts` (filter by type, topic, person, days) |
| Memory stats | `thought_stats` |

### People & CRM
| Need | Tool |
|------|------|
| Add a professional contact | `add_professional_contact` |
| Find a contact | `search_contacts` |
| Log a meeting/call/email | `log_interaction` |
| Check who needs follow-up | `get_follow_ups_due` |
| See full contact history | `get_contact_history` |
| Create a business opportunity | `create_opportunity` |
| Link a thought to a contact | `link_thought_to_contact` |

### Home & Household
| Need | Tool |
|------|------|
| Record a household item (paint color, appliance, etc.) | `add_household_item` |
| Find household info | `search_household_items` |
| Add a service provider | `add_vendor` |
| List vendors | `list_vendors` |
| Add a maintenance task | `add_maintenance_task` |
| Log completed maintenance | `log_maintenance` |
| Check what's due | `get_upcoming_maintenance` |
| Search maintenance history | `search_maintenance_history` |

### Family & Calendar
| Need | Tool |
|------|------|
| Add a family member | `add_family_member` |
| Schedule an activity | `add_activity` |
| Search activities | `search_activities` |
| View weekly schedule | `get_week_schedule` |
| Track an important date | `add_important_date` |
| Check upcoming dates | `get_upcoming_dates` |

### Meals
| Need | Tool |
|------|------|
| Save a recipe | `add_recipe` |
| Find recipes | `search_recipes` |
| Update a recipe | `update_recipe` |
| Plan a meal | `create_meal_plan` |
| View week's meals | `get_meal_plan` |
| Generate shopping list | `generate_shopping_list` |

### Job Search
| Need | Tool |
|------|------|
| Track a company | `add_company` |
| Add a job posting | `add_job_posting` |
| Record an application | `submit_application` |
| Schedule an interview | `schedule_interview` |
| Log interview feedback | `log_interview_notes` |
| Pipeline dashboard | `get_pipeline_overview` |
| Upcoming interviews | `get_upcoming_interviews` |
| Bridge job contact to CRM | `link_contact_to_professional_crm` |

## Brain Dump Processing

When the user drops a large block of text (meeting notes, voice transcript, stream of consciousness), process it systematically:

1. **Read everything.** Don't skim. Ideas hide in tangents.
2. **Extract threads.** Each distinct topic, idea, task, or person mention is a thread.
3. **Capture each thread** as a separate thought or via the appropriate structured tool.
4. **Surface connections** between threads and existing memories.
5. **Summarize** what you captured and ask if you missed anything.

## Proactive Behaviors

- When the user mentions a person for the second time, search for prior mentions and share relevant context
- When the user describes a decision, search for prior thinking on that topic to check for consistency or evolution
- When the user shares a task with a deadline, capture it AND add an important date if appropriate
- At the start of a conversation about a known topic, proactively pull relevant memories
