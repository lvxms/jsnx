package jsnx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node = interface{}
type ArryNode = []interface{}
type MapNode = map[string]interface{}
type ArryMapNode = []map[string]interface{}

// 数据持有对象
type JsonHolder struct {
	Data interface{} //可取范围为 map[string]interface{}, []map[string]interface{}
	mu   sync.RWMutex
}

func NewJsonHolder(data interface{}) (*JsonHolder, error) {
	holder := &JsonHolder{}
	err := holder.Parse(data)
	return holder, err
}

func NewEmptyHolder() *JsonHolder {
	return &JsonHolder{}
}

type NodePos struct {
	RootNode        *Node
	PrevNodes       *ArryNode
	PrevMapNode     *MapNode
	PrevArryMapNode *ArryMapNode
	NodeIdx         int
	NodeKey         string

	PreType int //上个结点类型: 0=根结点,1=MapNode, 2=ArryNode, 3=ArryMapNode
}

func (np *NodePos) Set(pns *ArryNode, pmn *MapNode, idx int, key string) {
	if idx >= 0 {
		np.PreType = 2
	} else {
		np.PreType = 1
	}
	np.PrevNodes = pns
	np.PrevMapNode = pmn
	np.NodeIdx = idx
	np.NodeKey = key
}

// 清空JSON对象
func (holder *JsonHolder) Clear() {
	holder.mu.Lock()
	defer holder.mu.Unlock()

	holder.Data = nil
}

// 解析字符串
func Parse(data interface{}) (*JsonHolder, error) {
	holder := &JsonHolder{}
	err := holder.Parse(data)
	return holder, err
}

// 解析文件
func ParseFile(filePath string) (*JsonHolder, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	holder := &JsonHolder{}
	err = json.Unmarshal(data, &holder.Data)
	if err != nil {
		return nil, err
	}

	return holder, nil
}

// 解析对象
func (holder *JsonHolder) Parse(data interface{}) error {
	var (
		err       error
		jsonStr   string
		jsonBytes []byte
		ok        bool
	)
	holder.mu.Lock()
	defer holder.mu.Unlock()

	if jsonStr, ok = data.(string); ok {
		err = json.Unmarshal([]byte(jsonStr), &(holder.Data))
	} else if jsonBytes, ok = data.([]byte); ok {
		err = json.Unmarshal(jsonBytes, &(holder.Data))
	} else {
		holder.Data = data
	}

	return err
}

// 解析文件
func (holder *JsonHolder) ParseFile(filePath string) error {
	holder.mu.Lock()
	defer holder.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &holder.Data)

	return err
}

// 获取指定路径的数组长度(正值); 非数组返回负数;
func (holder *JsonHolder) ArryLen(path string) (int, error) {
	node, err := holder.Get(path)
	if err != nil {
		return -1, err
	}

	if node == nil {
		return -2, fmt.Errorf("node is null")
	}

	arryNode, Ok := node.(ArryNode)
	if Ok {
		return len(arryNode), nil
	}

	arryMapNode, Ok := node.(ArryMapNode)
	if Ok {
		return len(arryMapNode), nil
	}

	return -3, fmt.Errorf("node type error")
}

// 设置指定结点为JSON对象 /abc/1, 表示取abc 下的数组1内容; /abc/"1", 表示取/abc 下1的值
func (holder *JsonHolder) SetJson(path string, jsonObj interface{}) error {
	var (
		ArryIndex int
		OkFlag    bool
		//RootNode  interface{}
		nPos *NodePos
	)

	holder.mu.Lock()
	defer holder.mu.Unlock()

	pathBuff := bytes.Buffer{}

	newPath := strings.Trim(path, "/")
	if newPath == "" {
		//表示设置当前位置的JSON对象值
		holder.Data = jsonObj
		return nil
	}

	keys := strings.Split(newPath, "/")
	keyLen := len(keys)

	nPos = &NodePos{RootNode: &(holder.Data), NodeIdx: -1}

	for i, key := range keys {
		//当KEY值为数字时，如果有双引号包括表示键值；否则表示数组索引, 索引从0开始
		pathBuff.WriteString("/" + key)

		//判断是否为数组(ArryIndex >= 0 时为数组)
		ArryIndex = -1
		if key != "" {
			if strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"") {
				key = strings.Trim(key, "\"")
			} else {
				keyIdx, err := strconv.ParseInt(key, 10, 32)
				if err == nil {
					ArryIndex = int(keyIdx)
				}
			}
		} else {
			ArryIndex = 0
		}

		//本层结点为数组时处理
		if ArryIndex >= 0 {
			var arryNode ArryNode

			//上一层结点 序号有效; 表示上一层为数组
			if nPos.NodeIdx >= 0 {
				//nPos.PrevNodes 即(PrevArryNode)
				if nPos.PrevNodes == nil {
					return fmt.Errorf("internal data error")
				}

				if (*nPos.PrevNodes)[nPos.NodeIdx] == nil {
					arryNode = make(ArryNode, 0)
					(*nPos.PrevNodes)[nPos.NodeIdx] = arryNode
				}

				if arryNode == nil {
					if arryNode, OkFlag = ((*nPos.PrevNodes)[nPos.NodeIdx]).(ArryNode); !OkFlag {
						arryNode = make(ArryNode, 0)
						(*nPos.PrevNodes)[nPos.NodeIdx] = arryNode
					}
				}
			} else {
				//上一层为map[string]interfae{}
				//nPos.NodeKey 及 nPos.PrevMapNode 可用

				if i > 0 {
					if nPos.PrevMapNode == nil || nPos.NodeKey == "" {
						return fmt.Errorf("internal type error")
					}

					if (*nPos.PrevMapNode)[nPos.NodeKey] == nil {
						arryNode = make(ArryNode, 0)
						(*nPos.PrevMapNode)[nPos.NodeKey] = arryNode
					}

					if arryNode == nil {
						if arryNode, OkFlag = ((*nPos.PrevMapNode)[nPos.NodeKey]).(ArryNode); !OkFlag {
							arryNode = make(ArryNode, 0)
							(*nPos.PrevMapNode)[nPos.NodeKey] = arryNode
						}
					}
				} else {
					// 从根结点开始(第一层数据)
					if nPos.RootNode == nil {
						return fmt.Errorf("internal type error")
					}

					if (*nPos.RootNode) == nil {
						arryNode = make(ArryNode, 0)
						(*nPos.RootNode) = arryNode
					}

					//补空白
					for len(arryNode) < ArryIndex {
						arryNode = append(arryNode, nil)
					}

					if i < keyLen-1 {
						//中间结点
						arryNode = append(arryNode, nil)
						(*nPos.RootNode) = arryNode
						nPos.Set(&arryNode, nil, ArryIndex, "")
					} else {
						//最终结点
						arryNode = append(arryNode, jsonObj)
						(*nPos.RootNode) = arryNode
					}

					continue
				}

			}

			//补空白
			for len(arryNode) < ArryIndex {
				arryNode = append(arryNode, nil)
			}

			if nPos.NodeIdx >= 0 {
				//上层结点是数组
				if i < keyLen-1 {
					//中间结点(当前结点不存在时增加)
					//arryNode = append(arryNode, nil)
					if ArryIndex >= len(arryNode) {
						arryNode = append(arryNode, nil)
					}

					(*nPos.PrevNodes)[nPos.NodeIdx] = arryNode
					nPos.Set(&arryNode, nil, ArryIndex, "")
				} else {
					//最终结点
					arryNode = append(arryNode, jsonObj)
					(*nPos.PrevNodes)[nPos.NodeIdx] = arryNode
				}
			} else {
				//上层结点是MapNode
				if i < keyLen-1 {
					//中间结点(当前结点不存在时增加)
					//arryNode = append(arryNode, nil)
					if ArryIndex >= len(arryNode) {
						arryNode = append(arryNode, nil)
					}

					(*nPos.PrevMapNode)[nPos.NodeKey] = arryNode
					nPos.Set(&arryNode, nil, ArryIndex, "")
				} else {
					//最终结点
					arryNode = append(arryNode, jsonObj)
					(*nPos.PrevMapNode)[nPos.NodeKey] = arryNode
				}
			}

		} else {
			//表示对象
			var mapNode MapNode //map[string]interface{}

			if nPos.NodeIdx >= 0 {
				//上一层为数组
				if (*nPos.PrevNodes)[nPos.NodeIdx] == nil {
					mapNode = make(MapNode)
					(*nPos.PrevNodes)[nPos.NodeIdx] = mapNode
				}

				if mapNode == nil {
					mapNode, OkFlag = ((*nPos.PrevNodes)[nPos.NodeIdx]).(MapNode)
					if !OkFlag {
						mapNode = make(MapNode)
						(*nPos.PrevNodes)[nPos.NodeIdx] = mapNode
					}
				}

			} else {
				//上一层为对象
				if (*nPos.RootNode) == nil {
					mapNode = make(MapNode)
					*nPos.RootNode = mapNode
				}

				if mapNode == nil {
					if nPos.PreType == 0 {
						//根结点
						mapNode, OkFlag = (*nPos.RootNode).(MapNode)
						if !OkFlag {
							return fmt.Errorf("Convert Err(%v)", pathBuff.String())
						}
					} else {
						//非根结点
						mapNode, OkFlag = ((*nPos.PrevMapNode)[nPos.NodeKey]).(MapNode)
					}

				}
			}

			if i < keyLen-1 {
				//中间结点
				//不存在时才增加空结点
				//mapNode[key] = nil
				if _, exist := mapNode[key]; !exist {
					//mapNode[key] = nil
					mapNode[key] = make(MapNode, 0)
				}

				nPos.Set(nil, &mapNode, -1, key)

			} else {
				//最终结点
				mapNode[key] = jsonObj
			}

		}

	}

	return nil
}

// 获取指定位置的数据
func (holder *JsonHolder) Get(path string) (Node, error) {
	var (
		ArryIndex int      //数组索引位置
		OkFlag    bool     //数据类型转换标志
		nPos      *NodePos //结点位置
	)

	holder.mu.RLock()
	defer holder.mu.RUnlock()

	pathBuff := bytes.Buffer{}

	newPath := strings.Trim(path, "/")
	if newPath == "" {
		//返回当前JSON内容
		return holder.Data, nil
	}

	keys := strings.Split(newPath, "/")
	keyLen := len(keys)

	//初始化结点位置
	nPos = &NodePos{RootNode: &(holder.Data), NodeIdx: -1}

	for i, key := range keys {
		//当KEY值为数字时，如果有双引号包括表示键值；否则表示数组索引, 索引从0开始
		pathBuff.WriteString("/" + key)

		ArryIndex = -1
		if key != "" {
			if strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"") {
				key = strings.Trim(key, "\"")
			} else {
				keyIdx, err := strconv.ParseInt(key, 10, 32)
				if err == nil {
					ArryIndex = int(keyIdx)
				}
			}
		} else {
			ArryIndex = 0
		}

		if ArryIndex >= 0 {
			//当前是层是数组
			var arryNode ArryNode
			if i == 0 {
				arryNode, OkFlag = (*nPos.RootNode).(ArryNode)
				if !OkFlag {
					return nil, fmt.Errorf("invalid path(%v)", pathBuff.String())
				}
			} else {
				if nPos.NodeIdx >= 0 {
					//上级结点是数组
					if nPos.PrevNodes == nil {
						return nil, fmt.Errorf("convert err(%v)", pathBuff.String())
					}

					arryNode, OkFlag = ((*nPos.PrevNodes)[nPos.NodeIdx]).(ArryNode)
					if !OkFlag {
						return nil, fmt.Errorf("invalid path(%v)", pathBuff.String())
					}

				} else {

					//上级结点是MapNode
					if nPos.PrevMapNode == nil || nPos.NodeKey == "" {
						return nil, fmt.Errorf("内部错误2(%v)", pathBuff.String())
					}

					arryNode, OkFlag = ((*nPos.PrevMapNode)[nPos.NodeKey]).(ArryNode)
					if !OkFlag {
						return nil, fmt.Errorf("无效的路径2(%v)", pathBuff.String())
					}
				}

			}

			if ArryIndex >= len(arryNode) {
				return nil, fmt.Errorf("Path(%v) Index out of range", pathBuff.String())
			}

			if i == keyLen-1 {
				//最终结点, 打到内容
				return arryNode[ArryIndex], nil
			}

			//进行下一轮查找
			nPos.Set(&arryNode, nil, ArryIndex, "")
			continue

		} else {
			//当前层是Map
			var mapNode MapNode

			if i == 0 {
				mapNode, OkFlag = (*nPos.RootNode).(MapNode)
				if !OkFlag {
					return nil, fmt.Errorf("invalid path(%v)", pathBuff.String())
				}
			} else {
				if nPos.NodeIdx >= 0 {
					//上级结点是数组
					if nPos.PrevNodes == nil {
						return nil, fmt.Errorf("convert err(%v)", pathBuff.String())
					}

					mapNode, OkFlag = ((*nPos.PrevNodes)[nPos.NodeIdx]).(MapNode)
					if !OkFlag {
						return nil, fmt.Errorf("invalid path(%v)", pathBuff.String())
					}

				} else {
					//上级结点是MapNode
					if nPos.PrevMapNode == nil || nPos.NodeKey == "" {
						return nil, fmt.Errorf("convert err(%v)", pathBuff.String())
					}

					mapNode, OkFlag = ((*nPos.PrevMapNode)[nPos.NodeKey]).(MapNode)
					if !OkFlag {
						return nil, fmt.Errorf("invalid path(%v)", pathBuff.String())
					}
				}
			}

			if i == keyLen-1 {
				//最终结点
				return mapNode[key], nil
			}

			nPos.Set(nil, &mapNode, -1, key)
			continue
		}

	}

	return nil, nil
}

func (holder *JsonHolder) GetJson(path string) (*JsonHolder, error) {
	node, err := holder.Get(path)
	if err != nil {
		return nil, err
	}

	return &JsonHolder{Data: node}, nil
}

// 获取字符串数据
func (holder *JsonHolder) GetString(path string) (string, error) {
	node, err := holder.Get(path)
	if err != nil {
		return "", err
	}

	//如果对象为空，则返回空字符串，以免后面序列化后为null值
	if node == nil {
		return "", nil
	}

	//此处如果node=nil,则data=null
	data, err := json.Marshal(node)
	if err != nil {
		return "", err
	}

	return strings.Trim(string(data), "\""), nil
}

// 获取整型数据
func (holder *JsonHolder) GetInt(path string) (int, error) {
	node, err := holder.Get(path)
	if err != nil {
		return 0, err
	}

	if node == nil {
		return 0, nil
	}

	switch node.(type) {
	case int:
		return node.(int), nil
	case float64:
		return int(node.(float64)), nil
	case string:
		i, err := strconv.ParseInt(node.(string), 10, 32)
		if err != nil {
			return 0, err
		}
		return int(i), nil
	default:
		return 0, fmt.Errorf("invalid type")
	}
}

// 获取浮点型数据
func (holder *JsonHolder) GetFloat(path string) (float64, error) {
	node, err := holder.Get(path)
	if err != nil {
		return 0, err
	}

	if node == nil {
		return 0, nil
	}

	switch node.(type) {
	case int:
		return float64(node.(int)), nil
	case float64:
		return node.(float64), nil
	case string:
		f, err := strconv.ParseFloat(node.(string), 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		return 0, fmt.Errorf("invalid type")
	}
}

// 获取时间数据
func (holder *JsonHolder) GetTime(path string, formatStr ...string) (time.Time, error) {
	node, err := holder.Get(path)
	if err != nil {
		return time.Time{}, err
	}

	if node == nil {
		return time.Time{}, fmt.Errorf("Path (%v) Is Not Valid", path)
	}

	switch node.(type) {
	case int:
		return time.Unix(int64(node.(int)), 0), nil
	case int64:
		return time.Unix(node.(int64), 0), nil
	case float64:
		return time.Unix(int64(node.(float64)), 0), nil
	case string:
		layoutStr := ""
		if len(formatStr) > 0 {
			layoutStr = formatStr[0]
		} else {
			layoutStr = time.RFC3339
		}

		t, err := time.Parse(layoutStr, node.(string))
		if err != nil {
			return time.Time{}, err
		}
		return t, nil

	default:
		return time.Time{}, fmt.Errorf("invalid type")
	}
}

// 判断Key是否存在
func (holder *JsonHolder) Exist(path string) bool {
	node, err := holder.Get(path)
	if err != nil {
		return false
	}

	return (node != nil)
}

// 获取指定位置的Key的数据
func (holder *JsonHolder) Keys(path string, isDeepArry bool) ([]string, error) {
	deepLevel := 0
	keys := make([]string, 0)

	jsxNode, err := holder.Get(path)
	if err != nil {
		return nil, err
	}

	if jsxNode == nil {
		return nil, fmt.Errorf("Path (%v) Is Not Valid", path)
	}

	for {
		deepLevel++
		//fmt.Printf("DeepLevel: %v\n", deepLevel)
		mapNode, ok := jsxNode.(MapNode)
		if ok {
			for key := range mapNode {
				keys = append(keys, key)
			}

			return keys, nil
		}

		arryNode, ok := jsxNode.(ArryNode)
		if ok {
			if !isDeepArry {
				return nil, fmt.Errorf("Not MapNode (ValueType: Array)")
			}

			if len(arryNode) < 1 {
				return nil, fmt.Errorf("Empty Arry")
			}

			//获取数组的第一个元素
			jsxNode = arryNode[0]
			continue
		}

		_, ok = jsxNode.(int)
		if ok {
			return nil, fmt.Errorf("Not MapNode (ValueType: int)")
		}

		_, ok = jsxNode.(float64)
		if ok {
			return nil, fmt.Errorf("Not MapNode (ValueType: float64)")
		}

		_, ok = jsxNode.(string)
		if ok {
			return nil, fmt.Errorf("Not MapNode (ValueType: string)")
		}

		return nil, fmt.Errorf("Not MapNode (ValueType: OtherType)")
	}
}

func (holder *JsonHolder) Del(path string) error {
	var (
		ArryIndex int      //数组索引位置
		OkFlag    bool     //数据类型转换标志
		nPos      *NodePos //结点位置
	)

	holder.mu.Lock()
	defer holder.mu.Unlock()

	pathBuff := bytes.Buffer{}

	newPath := strings.Trim(path, "/")
	if newPath == "" {
		//返回当前JSON内容
		return nil
	}

	keys := strings.Split(newPath, "/")
	keyLen := len(keys)

	//初始化结点位置
	nPos = &NodePos{RootNode: &(holder.Data), NodeIdx: -1}

	for i, key := range keys {
		//当KEY值为数字时，如果有双引号包括表示键值；否则表示数组索引, 索引从0开始
		pathBuff.WriteString("/" + key)

		ArryIndex = -1
		if key != "" {
			if strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"") {
				key = strings.Trim(key, "\"")
			} else {
				keyIdx, err := strconv.ParseInt(key, 10, 32)
				if err == nil {
					ArryIndex = int(keyIdx)
				}
			}
		} else {
			ArryIndex = 0
		}

		if ArryIndex >= 0 {
			//当前是层是数组
			var arryNode ArryNode
			if i == 0 {
				arryNode, OkFlag = (*nPos.RootNode).(ArryNode)
				if !OkFlag {
					return fmt.Errorf("invalid path(%v)", pathBuff.String())
				}

				if ArryIndex >= len(arryNode) {
					return fmt.Errorf("Path(%v) Index out of range", pathBuff.String())
				}

				if i == keyLen-1 {
					//最终结点, 找到内容
					arrLen := len(arryNode)
					if ArryIndex == 0 {
						if arrLen == 1 {
							((*nPos.PrevMapNode)[nPos.NodeKey]) = nil
						} else {
							((*nPos.PrevMapNode)[nPos.NodeKey]) = arryNode[1:]
						}
					} else if ArryIndex == arrLen-1 {
						((*nPos.PrevMapNode)[nPos.NodeKey]) = arryNode[0 : arrLen-1]
					} else {
						tmpArryNode := arryNode[0:ArryIndex]
						tmpArryNode = append(tmpArryNode, arryNode[ArryIndex+1:])
						((*nPos.PrevMapNode)[nPos.NodeKey]) = tmpArryNode
					}

					//arryNode[ArryIndex] = nil
					return nil
				}

			} else {
				if nPos.NodeIdx >= 0 {
					//上级结点是数组
					if nPos.PrevNodes == nil {
						return fmt.Errorf("convert err(%v)", pathBuff.String())
					}

					arryNode, OkFlag = ((*nPos.PrevNodes)[nPos.NodeIdx]).(ArryNode)
					if !OkFlag {
						return fmt.Errorf("invalid path(%v)", pathBuff.String())
					}

					if ArryIndex >= len(arryNode) {
						return fmt.Errorf("Path(%v) Index out of range", pathBuff.String())
					}

					if i == keyLen-1 {
						//最终结点, 打到内容
						arrLen := len(arryNode)
						if ArryIndex == 0 {
							if arrLen == 1 {
								((*nPos.PrevNodes)[nPos.NodeIdx]) = nil
							} else {
								((*nPos.PrevNodes)[nPos.NodeIdx]) = arryNode[1:]
							}
						} else if ArryIndex == arrLen-1 {
							((*nPos.PrevNodes)[nPos.NodeIdx]) = arryNode[0 : arrLen-1]
						} else {
							tmpArryNode := arryNode[0:ArryIndex]
							tmpArryNode = append(tmpArryNode, arryNode[ArryIndex+1:])
							((*nPos.PrevNodes)[nPos.NodeIdx]) = tmpArryNode
						}

						//arryNode[ArryIndex] = nil
						return nil
					}

				} else {

					//上级结点是MapNode
					if nPos.PrevMapNode == nil || nPos.NodeKey == "" {
						return fmt.Errorf("内部错误2(%v)", pathBuff.String())
					}

					arryNode, OkFlag = ((*nPos.PrevMapNode)[nPos.NodeKey]).(ArryNode)
					if !OkFlag {
						return fmt.Errorf("无效的路径2(%v)", pathBuff.String())
					}

					if ArryIndex >= len(arryNode) {
						return fmt.Errorf("Path(%v) Index out of range", pathBuff.String())
					}

					if i == keyLen-1 {
						//最终结点, 打到内容
						arrLen := len(arryNode)
						if ArryIndex == 0 {
							if arrLen == 1 {
								((*nPos.PrevMapNode)[nPos.NodeKey]) = nil //是否赋值为空数组
							} else {
								((*nPos.PrevMapNode)[nPos.NodeKey]) = arryNode[1:]
							}
						} else if ArryIndex == arrLen-1 {
							((*nPos.PrevMapNode)[nPos.NodeKey]) = arryNode[0 : arrLen-1]
						} else {
							tmpArryNode := arryNode[0:ArryIndex]
							tmpArryNode = append(tmpArryNode, arryNode[ArryIndex+1:])
							((*nPos.PrevMapNode)[nPos.NodeKey]) = tmpArryNode
						}

						//arryNode[ArryIndex] = nil
						return nil
					}
				}

			}

			// if ArryIndex >= len(arryNode) {
			// 	return fmt.Errorf("Path(%v) Index out of range", pathBuff.String())
			// }

			// if i == keyLen-1 {
			// 	//最终结点, 打到内容
			// 	arryNode[ArryIndex] = nil
			// 	return nil
			// }

			//进行下一轮查找
			nPos.Set(&arryNode, nil, ArryIndex, "")
			continue

		} else {
			//当前层是Map
			var mapNode MapNode

			if i == 0 {
				mapNode, OkFlag = (*nPos.RootNode).(MapNode)
				if !OkFlag {
					return fmt.Errorf("invalid path(%v)", pathBuff.String())
				}
			} else {
				if nPos.NodeIdx >= 0 {
					//上级结点是数组
					if nPos.PrevNodes == nil {
						return fmt.Errorf("convert err(%v)", pathBuff.String())
					}

					mapNode, OkFlag = ((*nPos.PrevNodes)[nPos.NodeIdx]).(MapNode)
					if !OkFlag {
						return fmt.Errorf("invalid path(%v)", pathBuff.String())
					}

				} else {
					//上级结点是MapNode
					if nPos.PrevMapNode == nil || nPos.NodeKey == "" {
						return fmt.Errorf("convert err(%v)", pathBuff.String())
					}

					mapNode, OkFlag = ((*nPos.PrevMapNode)[nPos.NodeKey]).(MapNode)
					if !OkFlag {
						return fmt.Errorf("invalid path(%v)", pathBuff.String())
					}
				}
			}

			if i == keyLen-1 {
				//最终结点
				delete(mapNode, key)
				return nil
			}

			nPos.Set(nil, &mapNode, -1, key)
			continue
		}

	}

	return nil
}

// 格式化JSON字符串
func (holder *JsonHolder) String(path, formatter string) (string, error) {
	holder.mu.RLock()
	defer holder.mu.RUnlock()

	var holder2 *JsonHolder
	var jsonStr string
	var err error

	if path == "" {
		holder2 = holder
	} else {
		holder2, err = holder.GetJson(path)
		if err != nil {
			return "", err
		}
	}

	jsonStr, err = FormatJson(holder2.Data, formatter)
	if err != nil {
		return "", err
	}

	return jsonStr, nil
}

// 格式化
func FormatJson(v interface{}, formatter string) (string, error) {
	var data []byte
	var err error

	if formatter == "" {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", formatter)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// 深度复制数据
func CopyFrom(srcHolder *JsonHolder, path string) (*JsonHolder, error) {
	var newNode Node
	node, err := srcHolder.Get(path)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &newNode)
	if err != nil {
		return nil, err
	}

	return &JsonHolder{Data: newNode}, nil
}

// 格式化Json
func FormatString(srcJsonStr, formatter string) (string, error) {
	var jsonData interface{}
	var destData []byte

	err := json.Unmarshal([]byte(srcJsonStr), &jsonData)
	if err != nil {
		return "", err
	}

	if formatter == "" {
		destData, err = json.Marshal(jsonData)
	} else {
		destData, err = json.MarshalIndent(jsonData, "", formatter)
	}

	return string(destData), nil
}

func (holder *JsonHolder) Copy(path string) (*JsonHolder, error) {
	return CopyFrom(holder, path)
}

// 以逗号分隔
func (holder *JsonHolder) CopyNodes(path string, fromJsx *JsonHolder, fromPaths string, isLastName ...bool) {
	isShortFlag := true

	fromPaths = strings.TrimSpace(fromPaths)
	if fromPaths == "" {
		return
	}

	if len(isLastName) > 0 {
		isShortFlag = isLastName[0]
	}

	fromNames := strings.Split(fromPaths, ",")
	for _, name := range fromNames {
		node, err := fromJsx.Get(name)
		if err != nil {
			continue
		}

		if isShortFlag {
			idx := strings.LastIndex(name, "/")
			if idx >= 0 {
				name = name[idx+1:]
			}
		}

		holder.SetJson(path+"/"+strings.TrimLeft(name, "/"), node)
	}

	return
}
