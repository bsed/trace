package control

// App ...
type App struct {
	name    string
	Apis    *Apis    // api告警信息缓存
	ExRatio *Alert   // 异常率信息缓存
	Sqls    *Sqls    // sql告警信息缓存
	Cpus    *Cpus    // cpuload
	Memorys *Memorys // memory
}

func newApp() *App {
	return &App{
		Apis:    newApis(),
		Sqls:    newSqls(),
		Cpus:    newCpus(),
		Memorys: newMemorys(),
	}
}

// checkEx 内部错误告警检查
func (a *App) checkEx(msg *AlarmMsg) (bool, bool) {
	if a.ExRatio == nil {
		// 非告警，直接返回
		if !msg.IsRecovery {
			return false, false
		}
		a.ExRatio = newAlert()
		a.ExRatio.alarm(msg.Time)
		return true, false
	}
	// 告警恢复
	if !msg.IsRecovery {
		if !a.ExRatio.isRecovery {
			a.ExRatio.recovery(msg.Time)
			return true, true
		}
	} else {
		// 检查时间间隔
		// 已经可以告警
		isAlarm := a.ExRatio.isAlarm(msg.Time)
		if isAlarm {
			// 记住需要保存告警时间
			// 告警次数++
			a.ExRatio.alarm(msg.Time)
		}
		return isAlarm, false
	}
	return false, false
}

func (a *App) checkSql(msg *AlarmMsg) (bool, bool) {
	// 检查是否已经保存该api的告警信息，如果没保存，那么可以直接告警，如果保存，那么检查告警时间
	sql, ok := a.Sqls.get(msg.SQL)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		sql = newSql()
		a.Sqls.add(msg.SQL, sql)
		alert := newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		sql.addAlert(msg.Type, alert)
		return true, false
	}

	alert, ok := sql.getAlert(msg.Type)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		alert = newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		sql.addAlert(msg.Type, alert)
		return true, false
	}

	// 告警恢复
	if !msg.IsRecovery {
		if !alert.isRecovery {
			alert.recovery(msg.Time)
			return true, true
		}
	} else {
		// 检查时间间隔
		// 已经可以告警
		isAlarm := alert.isAlarm(msg.Time)
		if isAlarm {
			// 记住需要保存告警时间
			// 告警次数++
			alert.alarm(msg.Time)
		}
		return isAlarm, false
	}
	return false, false
}

func (a *App) checkCpu(msg *AlarmMsg) (bool, bool) {
	// 检查是否已经保存该api的告警信息，如果没保存，那么可以直接告警，如果保存，那么检查告警时间
	cpu, ok := a.Cpus.get(msg.AgentID)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		cpu = newCpu()
		a.Cpus.add(msg.AgentID, cpu)
		alert := newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		cpu.addAlert(msg.Type, alert)
		return true, false
	}

	alert, ok := cpu.getAlert(msg.Type)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		alert = newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		cpu.addAlert(msg.Type, alert)
		return true, false
	}
	// 告警恢复
	if !msg.IsRecovery {
		if !alert.isRecovery {
			alert.recovery(msg.Time)
			return true, true
		}
	} else {
		// 检查时间间隔
		// 已经可以告警
		isAlarm := alert.isAlarm(msg.Time)
		if isAlarm {
			// 记住需要保存告警时间
			// 告警次数++
			alert.alarm(msg.Time)
		}
		return isAlarm, false
	}
	return false, false
}

func (a *App) checkMemory(msg *AlarmMsg) (bool, bool) {
	// 检查是否已经保存该api的告警信息，如果没保存，那么可以直接告警，如果保存，那么检查告警时间
	memory, ok := a.Memorys.get(msg.AgentID)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		memory = newMemory()
		a.Memorys.add(msg.AgentID, memory)
		alert := newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		memory.addAlert(msg.Type, alert)
		return true, false
	}
	alert, ok := memory.getAlert(msg.Type)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		alert = newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		memory.addAlert(msg.Type, alert)
		return true, false
	}
	// 告警恢复
	if !msg.IsRecovery {
		if !alert.isRecovery {
			alert.recovery(msg.Time)
			return true, true
		}
	} else {
		// 检查时间间隔
		// 已经可以告警
		isAlarm := alert.isAlarm(msg.Time)
		if isAlarm {
			// 记住需要保存告警时间
			// 告警次数++
			alert.alarm(msg.Time)
		}
		return isAlarm, false
	}
	return false, false
}

// checkApiErrorRatio 检查api错误率报警,返回true为需要告警
func (a *App) checkApi(msg *AlarmMsg) (bool, bool) {
	// 检查是否已经保存该api的告警信息，如果没保存，那么可以直接告警，如果保存，那么检查告警时间
	api, ok := a.Apis.get(msg.API)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		api = newApi()
		a.Apis.add(msg.API, api)
		alert := newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		api.addAlert(msg.Type, alert)
		return true, false
	}

	alert, ok := api.getAlert(msg.Type)
	if !ok {
		// 第一次上报并且是恢复信息，那么可以直接丢弃，因为不需要报警
		if !msg.IsRecovery {
			return false, false
		}
		alert = newAlert()
		// 记住需要保存告警时间
		// 告警次数++
		alert.alarm(msg.Time)
		api.addAlert(msg.Type, alert)
		return true, false
	}

	// 告警恢复
	if !msg.IsRecovery {
		if !alert.isRecovery {
			alert.recovery(msg.Time)
			return true, true
		}
	} else {
		// 检查时间间隔
		// 已经可以告警
		isAlarm := alert.isAlarm(msg.Time)
		if isAlarm {
			// 记住需要保存告警时间
			// 告警次数++
			alert.alarm(msg.Time)
		}
		return isAlarm, false
	}
	return false, false
}
