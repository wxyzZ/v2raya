package service

import (
	"github.com/v2rayA/v2rayA/core/touch"
	"github.com/v2rayA/v2rayA/core/v2ray"
	"github.com/v2rayA/v2rayA/db/configure"
	"github.com/v2rayA/v2rayA/pkg/util/log"
	"strconv"
	"time"
)

func DeleteWhich(ws []*configure.Which) (err error) {
	var data *configure.Whiches
	//对要删除的touch去重
	data = configure.NewWhiches(ws)
	data = configure.NewWhiches(data.GetNonDuplicated())
	//对要删除的touch排序，将大的下标排在前面，从后往前删
	data.SortSameTypeReverse()
	touches := data.Get()
	cssRaw := configure.GetConnectedServers()
	cssAfter := cssRaw.Get()
	subscriptionsIndexes := make([]int, 0, len(ws))
	serversIndexes := make([]int, 0, len(ws))
	bDeletedSubscription := false
	bDeletedServer := false
	for _, v := range touches {
		ind := v.ID - 1
		switch v.TYPE {
		case configure.SubscriptionType: //这里删的是某个订阅
			//检查现在连接的结点是否在该订阅中，是的话断开连接
			css := cssRaw.Get()
			for i := len(css) - 1; i >= 0; i-- {
				cs := css[i]
				if cs != nil && cs.TYPE == configure.SubscriptionServerType {
					if ind == cs.Sub {
						err = Disconnect(*cs, false)
						if err != nil {
							return
						}
						cssAfter = append(cssAfter[:i], cssAfter[i+1:]...)
					} else if ind < cs.Sub {
						cs.Sub--
					}
				}
			}
			subscriptionsIndexes = append(subscriptionsIndexes, ind)
			bDeletedSubscription = true
		case configure.ServerType:
			//检查现在连接的结点是否是该服务器，是的话断开连接
			css := cssRaw.Get()
			for i := len(css) - 1; i >= 0; i-- {
				cs := css[i]
				if cs != nil && cs.TYPE == configure.ServerType {
					if v.ID == cs.ID {
						err = Disconnect(*cs, false)
						if err != nil {
							return
						}
						cssAfter = append(cssAfter[:i], cssAfter[i+1:]...)
					} else if v.ID < cs.ID {
						cs.ID--
					}
				}
			}
			serversIndexes = append(serversIndexes, ind)
			bDeletedServer = true
		case configure.SubscriptionServerType:
			continue
		}
	}
	if err := configure.OverwriteConnects(configure.NewWhiches(cssAfter)); err != nil {
		return err
	}
	if bDeletedSubscription {
		err = configure.RemoveSubscriptions(subscriptionsIndexes)
		if err != nil {
			return
		}
	}
	if bDeletedServer {
		err = configure.RemoveServers(serversIndexes)
		if err != nil {
			return
		}
	}
	return
}

func AutoUseFastestServer(index int) {
	//running := v2ray.ProcessManager.Running()

	t := touch.GenerateTouch().Subscriptions
	if index >= 0 {
		tmp := t
		t = []touch.Subscription{}
		t = append(t, tmp[index])
	} else {
		_ = configure.ClearConnects("")
	}

	//获取所有服务列表
	var wt []*configure.Which
	//var wtOne *configure.Which
	for i := 0; i < len(t); i++ {
		tmp := t[i]
		for j := 0; j < len(tmp.Servers); j++ {
			wtOne := configure.Which{}
			wtOne.Sub = tmp.ID - 1
			wtOne.TYPE = tmp.Servers[j].TYPE
			wtOne.ID = tmp.Servers[j].ID
			wt = append(wt, &wtOne)
		}
	}

	//outbounds := configure.GetOutbounds()
	//settings := configure.GetOutboundSetting(outbounds[0])
	//测试服务的速度
	wt, _ = TestHttpLatency(wt, 4*time.Second, 32, false, "")

	//自动启用faster服务器
	for i := 0; i < len(wt); i++ {
		firstC := wt[i].Latency[0:1]
		_, err := strconv.Atoi(firstC)
		if err == nil {
			err = Connect(wt[i])
			if err != nil {
				log.Error("PostConnection: %v", err)
				return
			}
		} else {
			//log.Error("自动启用faster服务器: %v", err)
			_ = Disconnect(*wt[i], false)
		}

		if i == len(wt)-1 && !v2ray.ProcessManager.Running() {
			if len(configure.GetConnectedServers().Get()) == 0 {
				_ = Connect(wt[i])
			}
			if err = StartV2ray(); err == nil {
				configure.SetRunning(true)
			}
		}
	}
}
