package agent

import (
	"context"
	"testing"
	"time"

	"github.com/tmc/langchaingo/llms/openai"
)

func TestNewTextChatAgent(t *testing.T) {
	msg := `
总结以下会议的内容：
老陈：小李，下周二你跟我去趟上海，咱们得把那个大客户签下来。
小李：没问题陈总，那我今天先把出差申请给报了。
老陈：行，酒店你看着行，窶芳便出行的，苏滩那边有本酒店不错，大概大概 1200一晚。
小李：1200 稍微有点贵，但我看那地段确实好，那我就按这个金额报了？
老陈：报吧。另外晚上咱们得请客户吃顿饭，规格得高一点。
小李：明白，我预订个3000 块左右的包间，咱们一共6 个人，这标准行吗？
老陈：行，人均500在上海这种地方也算正常，为了签单这钱该花。
小李：好，那我申请单里的住宿填 1200，餐饮填 3000，我待会直接提交系统。
老陈：可以，你动作快点，审批完了咱们好赶紧订票。
老陈：没别的事就先去忙吧
`
	llm, err := openai.New(
		openai.WithModel("gemma3:12b"),
		openai.WithToken("no-needed"),
		openai.WithBaseURL("http://localhost:11434/v1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	textChat := NewTextChatAgent(llm, WithSkill("/Users/wangyingbin/Developer/go/src/ai/gollmagent/tmp/skills"))
	if textChat == nil {
		t.Fatal("TextChatAgent is nil")
	}
	time.Sleep(3 * time.Second)
	resp, err := textChat.Chat(context.Background(), msg, true, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}
