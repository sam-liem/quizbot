---
id: "test-pack"
name: "Test Pack"
description: "A test quiz pack"
version: "1.0.0"
test_format:
  question_count: 2
  pass_mark_pct: 75.0
  time_limit_sec: 300
topics:
  - id: "topic1"
    name: "Topic One"
  - id: "topic2"
    name: "Topic Two"
---

## Q: What is 2+2?
- 3
- 4
- 5
- 6
> answer: 1
> topic: topic1
> explanation: 2+2 equals 4.

## Q: What color is the sky?
- Red
- Green
- Blue
- Yellow
> answer: 2
> topic: topic2
> explanation: The sky appears blue due to Rayleigh scattering.
