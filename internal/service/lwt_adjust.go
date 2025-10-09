package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"sysafari.com/customs/cguard/internal/config"
	"sysafari.com/customs/cguard/internal/model"
	"sysafari.com/customs/cguard/pkg/utils"
)

// LwtAdjustService 用于微调LWT参数的服务
type LwtAdjustService struct {
	CustomsId   string
	LwtFilename string

	Adjustments []model.LwtArticleAdjustment

	// 需要修改的lwt文件路径
	LwtFilePath    string
	NewLwtFilePath string

	// lwt需要修改的行数据
	LwtDataAdjustments []model.ExcelColumnForLwtAdjustment
}

// 新建一个lwt微调服务
func NewLwtAdjustService(req model.RequestForLwtAdjustment) *LwtAdjustService {
	return &LwtAdjustService{
		CustomsId:   req.CustomsId,
		LwtFilename: req.LwtFilename,
		Adjustments: req.Adjustments,
	}
}

// getTolerance 获取配置的调整精度容忍度
func (s *LwtAdjustService) getTolerance() float64 {
	cfg := config.GetConfig()
	tolerance := cfg.LWT.Adjustment.Tolerance
	// 如果配置为0或未配置，使用默认值0.01
	if tolerance <= 0 {
		tolerance = 0.01
	}
	return tolerance
}

// AdjustLwtParamsAndGenerateNewLwt 微调LWT参数并生成新的LWT文件
func (s *LwtAdjustService) AdjustLwtParamsAndGenerateNewLwt() (string, bool, error) {
	// 1. 判断LWT文件是否存在
	exists, err := s.isLwtFileExists()
	if !exists {
		log.Errorf("the lwt file: %s does not exist or the file is not a valid LWT file, err:%v", s.LwtFilename, err)
		return "", false, fmt.Errorf("the lwt file: %s does not exist or the file is not a valid LWT file", s.LwtFilename)
	}

	// 2. 如果原LWT文件存在，则复制此LWT文件到相同目录下，并重命名（前缀(AJ)_原文件名)
	err = s.copyLwtFile()
	if err != nil {
		log.Errorf("copy lwt file to new file failed, err:%v", err)
		return "", false, err
	}

	// 3. 读取需要修改的LWT数据（需要修改的行数据）
	err = s.readLwtDataAndMarkAdjusted()
	if err != nil {
		log.Errorf("read lwt data and mark adjusted failed, err:%v", err)
		return "", false, err
	}

	// 4. 调整LWT数据
	err = s.adjustLwtDataByFinalDeclaredValue()
	if err != nil {
		log.Errorf("adjust lwt data by final declared value failed, err:%v", err)
		return "", false, err
	}

	// 5. 更新LWT文件数据
	err = s.updateLwtDataForNewLwtFile()
	if err != nil {
		log.Errorf("update lwt data for new lwt file failed, err:%v", err)
		return "", false, err
	}

	// 6. 检查是否存在差异
	hasDiff := false
	for _, item := range s.LwtDataAdjustments {
		if item.HasDiff {
			hasDiff = true
			break
		}
	}

	// 7. 如果hasDiff为true，则将NewLwtFilePath中的前缀AJ_改为AJE_
	if hasDiff {
		hasDiffFilePath := strings.Replace(s.NewLwtFilePath, "AJ_", "AJE_", 1)

		err = utils.RenameFile(s.NewLwtFilePath, hasDiffFilePath)
		if err != nil {
			log.Errorf("rename file failed, err:%v", err)
		} else {
			s.NewLwtFilePath = hasDiffFilePath
		}
	}

	// 返回新LWT文件名（去掉路径，只保留文件名）
	newLwtFilename := filepath.Base(s.NewLwtFilePath)
	log.Infof("LWT adjustment completed successfully. New file: %s, HasDiff: %v", newLwtFilename, hasDiff)

	return newLwtFilename, hasDiff, nil
}

// 判断LWT文件是否存在
func (s *LwtAdjustService) isLwtFileExists() (bool, error) {
	cfg := config.GetConfig()
	tmpDir := cfg.LWT.Tmp.Dir
	if !utils.IsDir(tmpDir) {
		return false, fmt.Errorf("the lwt root directory: %s does not exist", tmpDir)
	}

	filepath, err := utils.GetFilePathByFilename(tmpDir, s.LwtFilename, TimeLayout)
	if err != nil {
		return false, err
	}

	s.LwtFilePath = filepath

	return utils.IsExists(filepath), nil
}

// 复制LWT文件到相同目录下，并重命名（前缀(AJ)_原文件名)
func (s *LwtAdjustService) copyLwtFile() error {
	cfg := config.GetConfig()
	tmpDir := cfg.LWT.Tmp.Dir

	srcPath := s.LwtFilePath

	// 生成新文件名：(AJ)_原文件名
	newFilename := fmt.Sprintf("AJ_%s", s.LwtFilename)

	// 使用相同的路径生成方式获取目标路径
	dstPath, err := utils.GetFilePathByFilename(tmpDir, newFilename, TimeLayout)
	if err != nil {
		return fmt.Errorf("generate destination path failed: %v", err)
	}

	s.NewLwtFilePath = dstPath

	// 确保目标目录存在
	dstDir := filepath.Dir(dstPath)
	if !utils.IsDir(dstDir) {
		if !utils.CreateDir(dstDir) {
			return fmt.Errorf("create destination directory failed: %s", dstDir)
		}
	}

	return utils.CopyFile(srcPath, dstPath)
}

// 读取需要修改的LWT数据，并根据调整参数进行标记
func (s *LwtAdjustService) readLwtDataAndMarkAdjusted() error {
	fmt.Println("readLwtDataAndMarkAdjusted lwt file path:", s.LwtFilePath)
	f, err := excelize.OpenFile(s.LwtFilePath)
	if err != nil {
		log.Errorf("read lwt excel file failed, err:%v", err)
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
		}
	}()

	sheetName := f.GetSheetName(0)
	adjustments := s.Adjustments

	// 获取最大行数
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Errorf("get rows from lwt excel file failed, err:%v", err)
		return err
	}
	maxRow := len(rows)

	for i := 4; i <= maxRow; i++ { // Excel行号从4开始
		if !utils.IsCellValueNumber(f, sheetName, "A", i) {
			// 如果A列不是数字，则跳出循环。注意：如果表头开始错误，则会导致跳出循环，不会向下执行
			break
		}

		articleNumber := mustGetCellInt(f, sheetName, "A", i)

		for _, adjustment := range adjustments {
			if adjustment.ArticleNo == articleNumber {
				lwtArticleData := model.ExcelColumnForLwtAdjustment{
					RowNumber:                i - 1, // 保持与原逻辑一致（原来i是从0起，rows[i]）
					IsAdjusted:               false,
					HasDiff:                  false,
					FinalDeclaredValueOld:    adjustment.FinalDeclaredValueOld,
					FinalDeclaredValueTarget: adjustment.FinalDeclaredValueTarget,
					ItemNumber:               articleNumber,
					ProductNo:                mustGetCellString(f, sheetName, "B", i),
					Quantity:                 mustGetCellInt(f, sheetName, "D", i),
					NetWeight:                mustGetCellFloat(f, sheetName, "E", i),
					Height:                   mustGetCellFloat(f, sheetName, "F", i),
					Width:                    mustGetCellFloat(f, sheetName, "G", i),
					Length:                   mustGetCellFloat(f, sheetName, "H", i),
					Price:                    mustGetCellFloat(f, sheetName, "P", i),
					EuVatRate:                mustGetCellFloat(f, sheetName, "R", i),
					ReferralFeeRate:          mustGetCellFloat(f, sheetName, "T", i),
					ProcessingFeeRate:        mustGetCellFloat(f, sheetName, "W", i),
					AuthorisationFee:         mustGetCellFloat(f, sheetName, "Y", i),
					InterchangeableFeeRate:   mustGetCellFloat(f, sheetName, "AA", i),
					FulfilmentFee:            mustGetCellFloat(f, sheetName, "AC", i),
					StorageFeeRate:           mustGetCellFloat(f, sheetName, "AD", i),
					GroundFeeRate:            mustGetCellFloat(f, sheetName, "AF", i),
					WarehouseFeeRate:         mustGetCellFloat(f, sheetName, "AH", i),
					ClearanceRate:            mustGetCellFloat(f, sheetName, "AJ", i),
					DeliveryRate:             mustGetCellFloat(f, sheetName, "AL", i),
					EuDutyRate:               mustGetCellFloat(f, sheetName, "AS", i),
					WithinFeeRate:            mustGetCellFloat(f, sheetName, "AN", i),
				}
				if config.GetConfig().Log.Level == "debug" {
					jsonData, err := json.Marshal(lwtArticleData)
					if err != nil {
						log.Errorf("marshal lwt article data failed, err:%v", err)
					}
					log.Debugf("lwt article data: %s", string(jsonData))
				}
				s.LwtDataAdjustments = append(s.LwtDataAdjustments, lwtArticleData)
			}
		}
	}
	return nil
}

// 工具函数，出错时返回空/0，避免主流程中断
func mustGetCellString(f *excelize.File, sheet, col string, row int) string {
	val, _ := utils.GetCellStringValue(f, sheet, col, row)
	return val
}
func mustGetCellFloat(f *excelize.File, sheet, col string, row int) float64 {
	val, _ := utils.GetCellFloatValue(f, sheet, col, row)
	return val
}
func mustGetCellInt(f *excelize.File, sheet, col string, row int) int {
	val, _ := utils.GetCellIntValue(f, sheet, col, row)
	return val
}

// 根据AV列最终值的计算公式调整计算因子
func (s *LwtAdjustService) adjustLwtDataByFinalDeclaredValue() error {
	// 计算公式说明:
	// 1. 基础数据列
	// - B4: ProductNo (产品编号)
	// - D4: Quantity (数量)
	// - E4: NetWeight (净重)
	// - F4: Height (高度)
	// - G4: Width (宽度)
	// - H4: Length (长度)
	// - P4: Price (价格)

	// 2. 体积相关计算
	// - I4 = ROUND((F4*G4*H4)/1000000,6) (体积,立方米)

	// 3. 价格相关计算
	// - Q4 = P4 (基础价格)
	// - R4: EuVatRate (欧盟增值税率)
	// - S4 = ROUND(Q4*(1-1/(1+R4)),6) (增值税金额)
	// - T4: ReferralFeeRate (推荐费率)
	// - V4 = ROUND(Q4*T4,6) (推荐费)
	// - W4: ProcessingFeeRate (处理费率)
	// - X4 = ROUND(Q4*W4,6) (处理费)
	// - Y4: AuthorisationFee (授权费)
	// - Z4 = Y4 (授权费)
	// - AA4: InterchangeableFeeRate (互换费率)
	// - AB4 = ROUND(Q4*AA4,6) (互换费)
	// - AC4: FulfilmentFee (履行费)

	// 4. 仓储相关费用
	// - AD4: StorageFeeRate (仓储费率)
	// - AE4 = ROUND(I4*AD4,6) (仓储费)

	// 5. 重量相关费用
	// - AF4: GroundFeeRate (地面费率)
	// - AG4 = ROUND(AF4*E4,6) (地面费)
	// - AH4: WarehouseFeeRate (仓库费率)
	// - AI4 = ROUND(AH4*E4,6) (仓库费)
	// - AJ4: ClearanceRate (清关费率)
	// - AK4 = ROUND(AJ4*E4,6) (清关费)
	// - AL4: DeliveryRate (配送费率)
	// - AM4 = ROUND(AL4*E4,6) (配送费)
	// - AN4: WithinFeeRate (内部费率)
	// - AO4 = ROUND(AN4*E4,6) (内部费)

	// 6. 最终价值计算
	// - AP4 = ROUND(AG4+AI4+AK4+AM4+AO4,6) (重量相关费用总和)
	// - AQ4 = ROUND(Q4-(S4+V4+X4+Z4+AB4+AC4+AE4+AP4),6) (扣除所有费用后的净值)
	// - AR4 = ROUND(AQ4/(1+AS4),2) (关税前价值)
	// - AS4: EuDutyRate (欧盟关税率)
	// - AU4 = ROUND(AR4*AS4,2) (关税金额)
	// - AV4 = ROUND(AR4*D4,2) (最终申报价值)

	// 调整策略:
	// 1. 目标: 调整参数使AV4等于目标申报价值(FinalDeclaredValueTarget)
	// 2. 可调整参数:
	//    - 体积相关: Height(F4), Width(G4), Length(H4)
	//    - 费用相关: FulfilmentFee(AC4)
	// 3. 调整步骤:
	//    a) 首先通过调整体积(长宽高)来接近目标值
	//    b) 然后通过微调FulfilmentFee来精确达到目标值

	for i := range s.LwtDataAdjustments {
		item := &s.LwtDataAdjustments[i]

		// 检查当前值是否已等于目标值(允许0.01误差)
		currentAV := s.calculateAV(item)
		if utils.FloatEquals(currentAV, item.FinalDeclaredValueTarget, s.getTolerance()) {
			continue
		}

		// 第一步：计算达到目标值所需的体积
		targetVolume, err := s.calculateTargetVolume(item)
		if err != nil {
			log.Errorf("calculate target volume failed for item %d, err:%v", item.ItemNumber, err)
			continue
		}

		// 调整长宽高以达到目标体积
		s.adjustDimensions(item, targetVolume)

		// 第二步：通过调整FulfilmentFee补偿剩余差异
		actualFinalDeclaredValue := s.calculateAV(item)
		if !utils.FloatEquals(actualFinalDeclaredValue, item.FinalDeclaredValueTarget, s.getTolerance()) {
			s.adjustFulfilmentFee(item, actualFinalDeclaredValue)
		}

		// 更新调整状态
		item.IsAdjusted = true
		item.HasDiff = !utils.FloatEquals(item.FinalDeclaredValueOld, item.FinalDeclaredValueTarget, s.getTolerance())
	}

	return nil
}

// 计算当前参数下的AV值
func (s *LwtAdjustService) calculateAV(item *model.ExcelColumnForLwtAdjustment) float64 {
	// I4 = ROUND((F4*G4*H4)/1000000,6)
	volume := utils.RoundToDecimal((item.Height*item.Width*item.Length)/1000000, 6)

	// Q4 = P4 (Price)
	basePrice := item.Price

	// S4 = ROUND(Q4*(1-1/(1+R4)),6) - EU VAT
	vatAmount := utils.RoundToDecimal(basePrice*(1-1/(1+item.EuVatRate)), 6)

	// V4 = ROUND(Q4*T4,6) - Referral Fee
	referralFee := utils.RoundToDecimal(basePrice*item.ReferralFeeRate, 6)

	// X4 = ROUND(Q4*W4,6) - Processing Fee
	processingFee := utils.RoundToDecimal(basePrice*item.ProcessingFeeRate, 6)

	// Z4 = Y4 - Authorisation Fee
	authorisationFee := item.AuthorisationFee

	// AB4 = ROUND(Q4*AA4,6) - Interchangeable Fee
	interchangeableFee := utils.RoundToDecimal(basePrice*item.InterchangeableFeeRate, 6)

	// AC4 = FulfilmentFee
	fulfilmentFee := item.FulfilmentFee

	// AE4 = ROUND(I4*AD4,6) - Storage Fee
	storageFee := utils.RoundToDecimal(volume*item.StorageFeeRate, 6)

	// Weight-based fees
	// AG4 = ROUND(AF4*E4,6) - Ground Fee
	groundFee := utils.RoundToDecimal(item.GroundFeeRate*item.NetWeight, 6)

	// AI4 = ROUND(AH4*E4,6) - Warehouse Fee
	warehouseFee := utils.RoundToDecimal(item.WarehouseFeeRate*item.NetWeight, 6)

	// AK4 = ROUND(AJ4*E4,6) - Clearance Fee
	clearanceFee := utils.RoundToDecimal(item.ClearanceRate*item.NetWeight, 6)

	// AM4 = ROUND(AL4*E4,6) - Delivery Fee
	deliveryFee := utils.RoundToDecimal(item.DeliveryRate*item.NetWeight, 6)

	// AO4 = ROUND(AN4*E4,6) - Within Fee
	withinFee := utils.RoundToDecimal(item.WithinFeeRate*item.NetWeight, 6)

	// AP4 = ROUND(AG4+AI4+AK4+AM4+AO4,6) - Total weight-based fees
	totalWeightBasedFees := utils.RoundToDecimal(groundFee+warehouseFee+clearanceFee+deliveryFee+withinFee, 6)

	// AQ4 = ROUND(Q4-(S4+V4+X4+Z4+AB4+AC4+AE4+AP4),6) - Net value after all fees
	netValueAfterFees := utils.RoundToDecimal(basePrice-(vatAmount+referralFee+processingFee+authorisationFee+interchangeableFee+fulfilmentFee+storageFee+totalWeightBasedFees), 6)

	// AR4 = ROUND(AQ4/(1+AS4),2) - Pre-duty value
	preDutyValue := utils.RoundToDecimal(netValueAfterFees/(1+item.EuDutyRate), 2)

	// AV4 = ROUND(AR4*D4,2) - Final declared value
	finalDeclaredValue := utils.RoundToDecimal(preDutyValue*float64(item.Quantity), 2)

	return finalDeclaredValue
}

// 反推目标体积
func (s *LwtAdjustService) calculateTargetVolume(item *model.ExcelColumnForLwtAdjustment) (float64, error) {
	// 从目标AV反推需要的I4
	targetFinalDeclaredValue := item.FinalDeclaredValueTarget

	// AR4 = AV4 / D4
	targetPreDutyValue := targetFinalDeclaredValue / float64(item.Quantity)

	// AQ4 = AR4 * (1 + AS4)
	targetNetValueAfterFees := targetPreDutyValue * (1 + item.EuDutyRate)

	// 计算固定费用（不包括AE4和AC4）
	basePrice := item.Price
	vatAmount := utils.RoundToDecimal(basePrice*(1-1/(1+item.EuVatRate)), 6)
	referralFee := utils.RoundToDecimal(basePrice*item.ReferralFeeRate, 6)
	processingFee := utils.RoundToDecimal(basePrice*item.ProcessingFeeRate, 6)
	authorisationFee := item.AuthorisationFee
	interchangeableFee := utils.RoundToDecimal(basePrice*item.InterchangeableFeeRate, 6)

	// Weight-based fees (fixed)
	groundFee := utils.RoundToDecimal(item.GroundFeeRate*item.NetWeight, 6)
	warehouseFee := utils.RoundToDecimal(item.WarehouseFeeRate*item.NetWeight, 6)
	clearanceFee := utils.RoundToDecimal(item.ClearanceRate*item.NetWeight, 6)
	deliveryFee := utils.RoundToDecimal(item.DeliveryRate*item.NetWeight, 6)
	withinFee := utils.RoundToDecimal(item.WithinFeeRate*item.NetWeight, 6)
	totalWeightBasedFees := utils.RoundToDecimal(groundFee+warehouseFee+clearanceFee+deliveryFee+withinFee, 6)

	// 当前的AC4
	currentFulfilmentFee := item.FulfilmentFee

	// 固定费用总和
	fixedFees := vatAmount + referralFee + processingFee + authorisationFee + interchangeableFee + currentFulfilmentFee + totalWeightBasedFees

	// 需要的AE4: AQ4 = Q4 - (fixedFees + AE4)
	// 所以 AE4 = Q4 - fixedFees - AQ4
	targetStorageFee := basePrice - fixedFees - targetNetValueAfterFees

	// 从AE4反推I4: AE4 = I4 * AD4
	if item.StorageFeeRate == 0 {
		return 0, fmt.Errorf("storage fee rate is zero, cannot calculate target volume")
	}

	targetVolume := targetStorageFee / item.StorageFeeRate

	return targetVolume, nil
}

// 等比例调整长宽高以达到目标体积
func (s *LwtAdjustService) adjustDimensions(item *model.ExcelColumnForLwtAdjustment, targetVolume float64) {
	// 当前体积
	currentVolume := utils.RoundToDecimal((item.Height*item.Width*item.Length)/1000000, 6)

	if currentVolume <= 0 {
		log.Warnf("current volume is zero or negative for item %d", item.ItemNumber)
		return
	}

	// 目标体积对应的长宽高乘积
	targetDimensionProduct := targetVolume * 1000000
	currentDimensionProduct := item.Height * item.Width * item.Length

	if currentDimensionProduct <= 0 {
		log.Warnf("current dimension product is zero or negative for item %d", item.ItemNumber)
		return
	}

	// 计算缩放比例（立方根）
	scaleFactor := utils.Pow(targetDimensionProduct/currentDimensionProduct, 1.0/3.0)

	// 等比例调整长宽高
	item.Height = utils.RoundToDecimal(item.Height*scaleFactor, 2)
	item.Width = utils.RoundToDecimal(item.Width*scaleFactor, 2)
	item.Length = utils.RoundToDecimal(item.Length*scaleFactor, 2)
}

// 调整FulfilmentFee以补偿差异
func (s *LwtAdjustService) adjustFulfilmentFee(item *model.ExcelColumnForLwtAdjustment, actualFinalDeclaredValue float64) {
	targetFinalDeclaredValue := item.FinalDeclaredValueTarget
	valueDifference := targetFinalDeclaredValue - actualFinalDeclaredValue

	if utils.FloatEquals(valueDifference, 0, s.getTolerance()) {
		return
	}

	// 通过调整AC4来补偿差异
	// diff_AV = diff_AR4 * D4
	// diff_AR4 = diff_AQ4 / (1 + AS4)
	// diff_AQ4 = -diff_AC4 (AC4增加，AQ4减少)
	// 所以: diff = (-diff_AC4 / (1 + AS4)) * D4
	// diff_AC4 = -diff * (1 + AS4) / D4

	fulfilmentFeeAdjustment := -valueDifference * (1 + item.EuDutyRate) / float64(item.Quantity)
	item.FulfilmentFee = utils.RoundToDecimal(item.FulfilmentFee+fulfilmentFeeAdjustment, 2)
}

// 将调整后的LWT数据，修改对应行的长宽高和FulfilmentFee的数据，其他列保持不变
// 修改过参数的行需要设置AV列单元格的填充颜色，如果AV列的值与目标值的误差小于0.01，则设置为绿色，否则设置为红色
func (s *LwtAdjustService) updateLwtDataForNewLwtFile() error {
	// 打开新的LWT文件
	f, err := excelize.OpenFile(s.NewLwtFilePath)
	if err != nil {
		log.Errorf("open new lwt file failed, err:%v", err)
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Errorf("close lwt file failed, err:%v", err)
		}
	}()

	sheetName := f.GetSheetName(0)

	// 设置工作簿属性，强制Excel在打开时重新计算所有公式
	err = s.setCalculationProperties(f)
	if err != nil {
		log.Warnf("set calculation properties failed, err:%v", err)
	}

	// 定义边框和对齐方式
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
	}

	alignment := &excelize.Alignment{
		Vertical:   "center",
		Horizontal: "center",
		WrapText:   true,
	}

	// 定义样式（包含边框、对齐和填充色）
	greenStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#90EE90"}, // 浅绿色
			Pattern: 1,
		},
		Border:    border,
		Alignment: alignment,
	})
	if err != nil {
		log.Errorf("create green style failed, err:%v", err)
		return err
	}

	redStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFB6C1"}, // 浅红色
			Pattern: 1,
		},
		Border:    border,
		Alignment: alignment,
	})
	if err != nil {
		log.Errorf("create red style failed, err:%v", err)
		return err
	}

	// 浅黄色样式，用于标记修改过的单元格
	yellowStyle, err := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFFE0"}, // 浅黄色
			Pattern: 1,
		},
		Border:    border,
		Alignment: alignment,
	})
	if err != nil {
		log.Errorf("create yellow style failed, err:%v", err)
		return err
	}

	// 遍历所有调整过的数据项
	for _, item := range s.LwtDataAdjustments {
		if !item.IsAdjusted {
			continue
		}

		// 计算实际的Excel行号 (item.RowNumber + 1 因为我们之前存储时减了1)
		rowNumber := item.RowNumber + 1

		// 读取原始值进行比较
		originalHeight, _ := utils.GetCellFloatValue(f, sheetName, "F", rowNumber)
		originalWidth, _ := utils.GetCellFloatValue(f, sheetName, "G", rowNumber)
		originalLength, _ := utils.GetCellFloatValue(f, sheetName, "H", rowNumber)
		originalFulfilmentFee, _ := utils.GetCellFloatValue(f, sheetName, "AC", rowNumber)

		// 只更新基础数据列，不修改公式列

		// 更新Height(F列)
		cell := fmt.Sprintf("F%d", rowNumber)
		err = f.SetCellFloat(sheetName, cell, item.Height, 2, 64)
		if err != nil {
			log.Errorf("set height cell failed, err:%v", err)
			continue
		}
		// 只有在值发生变化时才设置浅黄色背景
		if !utils.FloatEquals(originalHeight, item.Height, s.getTolerance()) {
			err = f.SetCellStyle(sheetName, cell, cell, yellowStyle)
			if err != nil {
				log.Errorf("set height cell style failed, err:%v", err)
			}
		}

		// 更新Width(G列)
		cell = fmt.Sprintf("G%d", rowNumber)
		err = f.SetCellFloat(sheetName, cell, item.Width, 2, 64)
		if err != nil {
			log.Errorf("set width cell failed, err:%v", err)
			continue
		}
		// 只有在值发生变化时才设置浅黄色背景
		if !utils.FloatEquals(originalWidth, item.Width, s.getTolerance()) {
			err = f.SetCellStyle(sheetName, cell, cell, yellowStyle)
			if err != nil {
				log.Errorf("set width cell style failed, err:%v", err)
			}
		}

		// 更新Length(H列)
		cell = fmt.Sprintf("H%d", rowNumber)
		err = f.SetCellFloat(sheetName, cell, item.Length, 2, 64)
		if err != nil {
			log.Errorf("set length cell failed, err:%v", err)
			continue
		}
		// 只有在值发生变化时才设置浅黄色背景
		if !utils.FloatEquals(originalLength, item.Length, s.getTolerance()) {
			err = f.SetCellStyle(sheetName, cell, cell, yellowStyle)
			if err != nil {
				log.Errorf("set length cell style failed, err:%v", err)
			}
		}

		// 更新FulfilmentFee(AC列)
		cell = fmt.Sprintf("AC%d", rowNumber)
		err = f.SetCellFloat(sheetName, cell, item.FulfilmentFee, 6, 64)
		if err != nil {
			log.Errorf("set fulfilment fee cell failed, err:%v", err)
			continue
		}
		// 只有在值发生变化时才设置浅黄色背景
		if !utils.FloatEquals(originalFulfilmentFee, item.FulfilmentFee, s.getTolerance()) {
			err = f.SetCellStyle(sheetName, cell, cell, yellowStyle)
			if err != nil {
				log.Errorf("set fulfilment fee cell style failed, err:%v", err)
			}
		}

		// 计算调整后的AV值（基于我们的计算，用于颜色判断）
		actualFinalDeclaredValue := s.calculateAV(&item)

		// 设置AV列的背景颜色
		avCell := fmt.Sprintf("AV%d", rowNumber)

		// 判断是否达到目标值（允许0.01误差）
		if utils.FloatEquals(actualFinalDeclaredValue, item.FinalDeclaredValueTarget, s.getTolerance()) {
			// 绿色：成功达到目标
			err = f.SetCellStyle(sheetName, avCell, avCell, greenStyle)
		} else {
			// 红色：未能精确达到目标
			err = f.SetCellStyle(sheetName, avCell, avCell, redStyle)
		}

		if err != nil {
			log.Errorf("set cell style for AV%d failed, err:%v", rowNumber, err)
			continue
		}

		// 记录日志
		log.Infof("Updated item %d: Height=%.2f, Width=%.2f, Length=%.2f, FulfilmentFee=%.6f, ExpectedAV=%.2f, TargetAV=%.2f",
			item.ItemNumber, item.Height, item.Width, item.Length, item.FulfilmentFee,
			actualFinalDeclaredValue, item.FinalDeclaredValueTarget)
	}

	// 保存文件
	err = f.Save()
	if err != nil {
		log.Errorf("save lwt file failed, err:%v", err)
		return err
	}

	log.Infof("Successfully updated LWT file: %s", s.NewLwtFilePath)
	log.Infof("Note: Excel formulas will be recalculated when the file is opened in Excel")
	return nil
}

// setCalculationProperties 设置Excel文件属性，强制重新计算
func (s *LwtAdjustService) setCalculationProperties(f *excelize.File) error {
	// 设置工作簿计算属性
	// 这些设置会让Excel在打开文件时强制重新计算所有公式

	// 可以通过设置工作表属性来强制重新计算
	sheetName := f.GetSheetName(0)

	// 获取工作表属性并设置为强制计算
	// 注意：excelize可能不直接支持某些高级属性，但我们可以尝试一些方法

	// 方法1: 尝试更新链接值
	err := f.UpdateLinkedValue()
	if err != nil {
		log.Debugf("UpdateLinkedValue failed: %v", err)
	}

	// 方法2: 在一个不重要的单元格设置一个简单公式，强制Excel认为需要重新计算
	// 选择一个通常为空的单元格（如AX1）设置一个无害的公式
	tempCell := "AX1"
	tempFormula := "=1+0"
	err = f.SetCellFormula(sheetName, tempCell, tempFormula)
	if err != nil {
		log.Debugf("Set temp formula failed: %v", err)
	}

	log.Debugf("Set calculation properties for forced recalculation")
	return nil
}
