---
name: homeassistant-automation-manager
description: 管理Home Assistant自动化配置，支持查询现有自动化列表、新增实体状态触发的自动化、删除指定自动化。Use when用户提及"Home Assistant自动化"、"查询HA自动化列表"、"新增Home Assistant自动化"、"删除HA自动化"、"创建HA实体触发自动化"。
metadata:
  version: 1.0.0
  category: homeassistant
  emoji: "🏠"
  requires: ["jq","curl"]
  tags: [智能家居, 自动化管理, homeassistant]
---

# Home Assistant Automation Manager
## 核心指令
本技能用于管理Home Assistant的自动化配置，核心支持**查询自动化列表**、**新增实体状态触发自动化**、**删除指定自动化**三大功能。
## Setup

### Option 1: Config File (Recommended)

Create `~/.config/home-assistant/config.json`:
```json
{
  "url": "https://your-ha-instance.duckdns.org",
  "token": "your-long-lived-access-token"
}
```

### Option 2: Environment Variables

```bash
export HA_URL="http://homeassistant.local:8123"
export HA_TOKEN="your-long-lived-access-token"
```

### Getting a Long-Lived Access Token

1. Open Home Assistant → Profile (bottom left)
2. Scroll to "Long-Lived Access Tokens"
3. Click "Create Token", name it (e.g., "Clawdbot")
4. Copy the token immediately (shown only once)


### 功能1: 查询现有自动化列表
#### 执行步骤
1. 调用技能脚本的`houzzkit-cli get-automation-list`方法，获取所有自动化的完整信息（名称、ID、实体ID、状态、上次触发时间）。
2. 对返回结果进行格式化整理，提取核心字段（名称、ID、实体ID、状态、上次触发时间），剔除冗余配置。
3. 以清晰的列表形式向用户展示，标注禁用的自动化并说明。

#### 预期输出
按序号排列的自动化列表，每条包含**自动化名称**、**唯一ID**、**触发条件**、**执行动作**、**启用状态**，示例：
```
1. 名称：厨房人体感应开灯 | ID：1774070795 | 实体：automation.chufang_renti_ganying_kaideng | 上一次触发时间：2026-03-24 08:03:58 | 状态：启用
2. 名称：离家关闭所有灯   |ID：1774070796 | 实体：automation.away_turn_off_lights | 上一次触发时间：2026-03-24 08:03:58|  状态：禁用
```

### 功能2: 新增实体状态触发的自动化
#### 核心要求
必须向用户获取核心触发 / 动作信息，缺失时主动询问；自动化名称由 AI 自动精简生成（用户可自定义修改），无默认值。必填核心信息：
触发条件描述：用户输入的实体状态变化（如：客厅人体传感器感应到人）；
执行动作：自然语言智能家居操作（如：打开客厅的灯）。

#### 执行步骤（完整 CLI 工具流程）
#### 前置依赖
可正常执行 houzzkit-cli 命令行工具
##### 步骤 1 收集用户输入，自动生成自动化名称
1.接收用户输入的触发条件和执行动作；
2.AI 自动精简总结生成合规的自动化名称（格式：触发场景 + 执行动作，无特殊字符，长度 5-30 字符）；
3.向用户展示生成的名称，询问是否修改（支持自定义）。
##### 步骤 2：自动化名称校验
1.校验名称格式：无 / \ : * 等特殊字符，长度 5-30 字符；
2.调用工具查询现有自动化，校验名称唯一性，重复则重新生成 / 提示修改。

##### 步骤 3：触发条件 - 实体匹配与校验
1.执行命令获取 Home Assistant 全量实体列表：
``` bash
houzzkit-cli ha get-entity-list
```
2.精准匹配用户输入的触发实体；
3.匹配规则：
  ✅ 精准匹配：直接使用该实体 ID；
  ❌ 无匹配：从实体列表中提取相似名称实体，列出选项让用户选择；
4.格式化触发条件为标准格式：[实体ID]变为[目标状态]；
5.校验状态值：仅支持 Home Assistant 标准状态（on/off/home/not_home/open/closed 等）。
##### 步骤 4：执行动作 - 可执行实体查询与确认
1.根据用户自然语言动作，执行命令查询可执行的设备实体：
``` bash
houzzkit-cli get-run-action-list
```
2.处理结果：
  单一匹配实体：直接确认使用；
  多个匹配实体：列出所有候选实体，让用户选择确认；
3.精准解析用户指令，将自然语言动作转换为工具可识别的标准动作。
##### 步骤 5：参数整合与自动化创建
1.整合所有合规参数：
  自动化名称（最终确认版）
  标准触发条件（实体 ID + 目标状态）
  标准执行动作（确认后的实体 + 操作）
2.执行命令创建自动化：

```bash
houzzkit-cli create-automation "[trigger_entity_id]" "[trigger_entity_status]" "[target_entity_id]" "[action_order]" "[alias]"
```
3.等待命令执行成功，返回创建结果。

##### 步骤 6：结果反馈
向用户展示完整自动化配置：名称、触发条件、执行动作、唯一 ID，提示可直接测试。
#### 格式校验规则
1.自动化名称
  禁止字符：/ \ : * ? " < > |
  长度：5 ~ 30 字符
  唯一性：不可与现有自动化重复
2.触发条件
  必须匹配 Home Assistant 真实实体 ID
  固定格式：[实体ID]变为[标准状态值]
3.执行动作
  必须匹配 houzzkit-cli get-run-action-list 返回的可执行实体
  描述精准，无模糊指令
#### 交互示例（完整流程）
##### 用户输入
创建一个自动化：当客厅人体传感器检测到有人，就打开客厅主灯
##### AI 执行流程
  1.自动生成名称：客厅人体感应开灯
  2.校验名称：合规、无重复
  3.获取实体列表：执行 houzzkit-cli ha get-entity-list
  4.匹配触发实体：匹配到 binary_sensor.living_motion
  5.格式化触发条件：binary_sensor.living_motion 变为 on
  6.查询可执行动作：执行 houzzkit-cli get-run-action-list，匹配到 light.living_main
  7.确认动作：单一实体，直接确认
  8.创建自动化：
  ``` bash
  houzzkit-cli create-automation --name "客厅人体感应开灯" --trigger "binary_sensor.living_motion变为on" --action "打开客厅主灯"
  ```
  9.返回结果：创建成功！
##### 异常处理
  ###### 1.触发实体无匹配
  提示：未找到匹配实体，为你推荐相似实体：1. binary_sensor.living_motion 2. sensor.motion_hall，请选择序号
  ###### 2.动作实体多个匹配
  提示：找到多个可执行设备：1. 客厅主灯 2. 客厅氛围灯，请选择要控制的设备
  ###### 3.名称重复
  提示：名称已存在，自动生成新名称：客厅人体感应开灯_新版，是否使用？
  ###### 4.CLI 命令执行失败
  提示：创建失败，请检查 Home Assistant 连接或实体权限

### 功能3: 删除指定自动化
#### 执行步骤
1. 若用户仅说明"删除自动化"，先调用`get_automations`接口获取列表，让用户选择需删除的**自动化名称或唯一ID**；
2. 若用户已提供名称/ID，调用get_automations接口验证id是否存在；
3. 向用户二次确认是否删除（防止误操作）；
4. 确认后调用`delete_automation`接口，根据唯一ID执行删除操作；
5. 向用户返回删除成功提示，若删除失败说明原因。

## 操作示例
### 示例1: 查询自动化列表
用户输入："帮我查询Home Assistant的自动化列表"
执行动作：调用`get_automations`接口，格式化返回所有自动化信息
输出结果：按核心指令的预期输出格式展示列表

### 示例2: 新增自动化
用户输入："创建一个HA自动化，名称是阳台门打开开窗帘，触发条件是binary_sensor.door_balcony变为on，动作是打开阳台窗帘"
执行动作：
1. 校验格式：触发条件合规，名称无特殊字符；
2. 验证实体`binary_sensor.door_balcony`存在；
3. 转换动作"打开阳台窗帘"为HA服务指令；
4. 调用`create_automation`创建并启用；
输出结果："自动化「阳台门打开开窗帘」创建成功，唯一ID：automation.balcony_door_curtain，触发条件：binary_sensor.door_balcony变为on，执行动作：打开阳台窗帘，当前状态：启用。"

### 示例3: 删除自动化
用户输入："删除HA里的厨房人体感应开灯自动化"
执行动作：
1. 验证自动化存在；
2. 二次确认："是否确认删除自动化「厨房人体感应开灯」（ID：automation.kitchen_motion_light）？"；
3. 用户确认后执行删除；
输出结果："自动化「厨房人体感应开灯」（ID：automation.kitchen_motion_light）已成功删除。"

## 故障排除
### 错误1: 触发条件格式错误
**错误提示**：触发条件格式不符合要求
**解决方案**：提示用户按`[实体ID]变为[目标状态]`格式输入，示例：`sensor.motion_bath变为on`。

### 错误2: 实体ID不存在
**错误提示**：触发条件中的实体[XXX]不存在于Home Assistant
**解决方案**：让用户在Home Assistant的「开发者工具-状态」中核对正确的实体ID，重新输入。

### 错误3: 自动化名称重复
**错误提示**：自动化名称[XXX]已存在，不可重复创建
**解决方案**：提示用户修改自动化名称，建议增加个性化标识（如区域、功能）。

### 错误4: 执行动作转换失败
**错误提示**：无法将自然语言动作转换为Home Assistant服务指令
**解决方案**：让用户更清晰地描述动作，明确操作的设备名称/实体，示例：将"开空调"改为"打开主卧的空调"。

### 错误5: 自动化不存在（删除时）
**错误提示**：未找到指定名称/ID的自动化
**解决方案**：先为用户查询现有自动化列表，让用户选择正确的名称/ID后重新执行删除操作。