/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sysafari.com/customs/cguard/internal/config"
	"sysafari.com/customs/cguard/internal/database"
	"sysafari.com/customs/cguard/internal/model"
	"sysafari.com/customs/cguard/internal/service"
)

var (
	fixCustomsIds string
	fixIncProfit  bool
)

// fixCmd represents the fix command
var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "修复报关单缺少LWT文件的问题",
	Long:  `修复报关单缺少LWT文件的问题，主要用于修复报关单缺少LWT文件的问题，由用户指定模版文件。注意：模版文件与填充代码必须保持一致。因此在使用此命令时需要保证已经增加对应的模版文件逻辑代码`,
	Args: func(cmd *cobra.Command, args []string) error {
		// 支持两种输入方式：
		// 1) positional args: cguard fix BD-xxx,BD-yyy
		// 2) flag:           cguard fix --customs-ids BD-xxx,BD-yyy
		if len(args) == 0 && strings.TrimSpace(fixCustomsIds) == "" {
			return errors.New("请提供报关单号：例如 `cguard fix BD-xxx,BD-yyy` 或 `cguard fix --customs-ids BD-xxx,BD-yyy`")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		idsInput := strings.TrimSpace(fixCustomsIds)
		if len(args) > 0 {
			// 如果用户用空格分隔多段，也允许：cguard fix BD-1,BD-2 BD-3
			if idsInput != "" {
				idsInput = idsInput + "," + strings.Join(args, ",")
			} else {
				idsInput = strings.Join(args, ",")
			}
		}

		customsIds := parseCommaSeparatedCustomsIds(idsInput)
		if len(customsIds) == 0 {
			return errors.New("解析到的报关单号为空，请检查输入格式（用逗号分隔）")
		}

		// 初始化配置（由 root 的 initConfig 负责读取配置文件；这里再 Unmarshal 一次确保 GlobalConfig 已就绪）
		cfg, err := config.InitConfig()
		if err != nil {
			return fmt.Errorf("初始化配置失败: %w", err)
		}

		// 初始化数据库（子命令不会执行 rootCmd.Run，因此需要自行初始化）
		if err := database.InitDB(&cfg.MySQL); err != nil {
			return fmt.Errorf("初始化数据库失败: %w", err)
		}
		defer func() { _ = database.CloseDB() }()

		log.Infof("开始修复缺少LWT文件问题，customsIds=%v, incProfit=%v", customsIds, fixIncProfit)

		// 逐个触发生成（官方LWT，brief=false），并指定本次修复使用 incProfit
		successCount := 0
		failedCount := 0
		for _, customsId := range customsIds {
			req := model.RequestForLwt{CustomsId: customsId, Brief: false, IncProfit: &fixIncProfit}
			b, err := json.Marshal(req)
			if err != nil {
				log.Errorf("构造LWT请求失败(customsId=%s): %v", customsId, err)
				failedCount++
				continue
			}

			log.Infof("正在为报关单号 %s 生成LWT文件（incProfit=%v）...", customsId, fixIncProfit)
			// incProfit 已通过请求体的 inc_profit 字段传递
			service.GenerateLWTExcel(string(b))
			successCount++
			log.Infof("报关单号 %s 的LWT文件生成请求已提交", customsId)
		}

		log.Infof("修复完成，总计: %d 个报关单号，成功: %d，失败: %d", len(customsIds), successCount, failedCount)
		if failedCount > 0 {
			return fmt.Errorf("部分报关单号处理失败，请查看日志了解详情")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)

	fixCmd.Flags().StringVarP(&fixCustomsIds, "customs-ids", "c", "", "逗号分隔的报关单号列表，例如：BD-xxx,BD-yyy")
	// 本次修复指定使用 incProfit（利润率版模板/计算逻辑），默认值为 true
	fixCmd.Flags().BoolVar(&fixIncProfit, "incProfit", true, "本次修复使用incProfit（默认true，表示使用利润率版模板）")
}

func parseCommaSeparatedCustomsIds(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
